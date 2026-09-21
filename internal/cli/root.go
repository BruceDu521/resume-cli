package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"resume-cli/internal/ai"
	"resume-cli/internal/app"
	"resume-cli/internal/cache"
	"resume-cli/internal/fileio"
	"resume-cli/internal/pdf"
)

type options struct {
	output, jd, lang, provider, model, baseURL, pipeline, jevModel, cacheDir, stats string
	mock, force                                                                     bool
	timeout                                                                         time.Duration
}
type recorder struct {
	mu     sync.Mutex
	calls  []ai.Usage
	hits   []string
	logger *slog.Logger
}

func (r *recorder) observe(u ai.Usage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, u)
	r.logger.Info("model call", "stage", u.Stage, "provider", u.Provider, "model", u.Model, "duration_ms", u.DurationMS, "attempts", u.Attempts, "input_tokens", u.Input, "output_tokens", u.Output, "json_repaired", u.Repaired)
}
func (r *recorder) hit(stage string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hits = append(r.hits, stage)
	r.logger.Info("cache hit", "stage", stage)
}
func (r *recorder) snapshot(elapsed time.Duration, mock bool, pipeline string, success bool) any {
	r.mu.Lock()
	defer r.mu.Unlock()
	return struct {
		Success  bool       `json:"success"`
		Elapsed  int64      `json:"elapsed_ms"`
		Mock     bool       `json:"mock"`
		Pipeline string     `json:"pipeline"`
		Calls    []ai.Usage `json:"calls"`
		Hits     []string   `json:"cache_hits"`
	}{success, elapsed.Milliseconds(), mock, pipeline, append([]ai.Usage{}, r.calls...), append([]string{}, r.hits...)}
}
func New(out, errOut io.Writer, getenv func(string) string) *cobra.Command {
	if getenv == nil {
		getenv = os.Getenv
	}
	o := options{}
	root := &cobra.Command{Use: "resume-cli", Short: "本地 PDF 简历解析与 AI 岗位匹配", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(errOut)
	f := root.PersistentFlags()
	f.StringVar(&o.output, "output", "", "保存结果到文件")
	f.BoolVar(&o.force, "force", false, "覆盖已有输出文件（不允许覆盖输入）")
	f.BoolVar(&o.mock, "mock", false, "使用离线合成演示模式")
	f.StringVar(&o.lang, "lang", "zh", "报告语言：zh / en")
	f.StringVar(&o.provider, "provider", getenv("RESUME_AI_PROVIDER"), "生成模型：gemini / deepseek / openai / kimi")
	f.StringVar(&o.model, "model", getenv("RESUME_AI_MODEL"), "覆盖生成模型 ID")
	f.StringVar(&o.baseURL, "base-url", getenv("RESUME_AI_BASE_URL"), "生成模型 HTTPS API 地址")
	f.StringVar(&o.pipeline, "pipeline", envDefault(getenv, "RESUME_AI_PIPELINE", "single"), "评分模式：single（默认）/ jev；hybrid 为 jev 别名")
	f.StringVar(&o.jevModel, "jev-model", envDefault(getenv, "RESUME_JEV_MODEL", "jev-1.13.0"), "可选 Jev 模型，仅 --pipeline jev 使用")
	f.StringVar(&o.cacheDir, "cache-dir", "", "显式启用 extract 或 Jev 结构化缓存")
	f.StringVar(&o.stats, "stats", "", "保存耗时及模型用量 JSON")
	f.DurationVar(&o.timeout, "timeout", 90*time.Second, "完整命令超时")
	for _, name := range []string{"parse", "extract", "score"} {
		name := name
		cmd := &cobra.Command{Use: name + " <pdf_path>", Args: cobra.ExactArgs(1)}
		if name == "score" {
			cmd.Flags().StringVar(&o.jd, "jd", "", "UTF-8 岗位描述文本")
			_ = cmd.MarkFlagRequired("jd")
		}
		cmd.RunE = func(cmd *cobra.Command, args []string) (runErr error) {
			start := time.Now()
			if o.timeout <= 0 || o.timeout > 10*time.Minute {
				return errors.New("timeout must be between 0 and 10 minutes")
			}
			if o.lang != "zh" && o.lang != "en" {
				return errors.New("lang must be zh or en")
			}
			if o.pipeline == "hybrid" {
				o.pipeline = "jev"
			}
			if o.pipeline != "single" && o.pipeline != "jev" {
				return errors.New("pipeline must be single or jev")
			}
			if name != "extract" && !(name == "score" && o.pipeline == "jev") && o.cacheDir != "" {
				return errors.New("--cache-dir applies only to extract or score --pipeline jev")
			}
			if err := checkPaths([]string{args[0], o.jd}, []string{o.output, o.stats}); err != nil {
				return err
			}
			level := slog.LevelInfo
			switch getenv("RESUME_LOG_LEVEL") {
			case "", "info":
			case "debug":
				level = slog.LevelDebug
			case "error":
				level = slog.LevelError
			case "warn":
				level = slog.LevelWarn
			default:
				return errors.New("invalid RESUME_LOG_LEVEL")
			}
			logger := slog.New(slog.NewTextHandler(errOut, &slog.HandlerOptions{Level: level}))
			rec := &recorder{logger: logger}
			for _, path := range []string{o.output, o.stats} {
				if path == "" {
					continue
				}
				if _, e := os.Lstat(path); e == nil && !o.force {
					return fmt.Errorf("output already exists: use --force to replace it")
				} else if e != nil && !errors.Is(e, os.ErrNotExist) {
					return e
				}
			}
			// Preserve timing and observed usage on failures as well as successes.
			// Do not include source text or untrusted provider errors in this file.
			defer func() {
				if o.stats == "" {
					return
				}
				stats, e := json.MarshalIndent(rec.snapshot(time.Since(start), o.mock, o.pipeline, runErr == nil), "", "  ")
				if e == nil {
					e = fileio.Write(o.stats, append(stats, '\n'), o.force)
				}
				if e != nil {
					runErr = errors.Join(runErr, fmt.Errorf("write stats: %w", e))
				}
			}()
			ctx, cancel := context.WithTimeout(cmd.Context(), o.timeout)
			defer cancel()
			s := app.Service{Parser: pdf.Parser{}, Cache: cache.Store{Dir: o.cacheDir}, Mock: o.mock, CacheHit: rec.hit}
			var jd string
			var err error
			if name == "score" {
				jd, err = fileio.Text(o.jd, 64<<10)
				if err != nil {
					return fmt.Errorf("JD: %w", err)
				}
			}
			if name != "parse" {
				if o.mock {
					logger.Warn("MOCK: synthetic fixture demonstration; no AI requests")
					m := ai.Mock{}
					s.Structurer = m
					s.Extractor = m
					if o.pipeline == "jev" {
						s.Matcher = m
					} else {
						s.Evaluator = m
					}
					s.Identity = "mock-v1"
				} else {
					g, err := remote(o, getenv)
					if err != nil {
						return err
					}
					st := ai.Structurer{Generator: g, Observe: rec.observe}
					s.Structurer = st
					s.Extractor = st
					s.Identity = g.Identity()
					if name == "score" && o.pipeline == "jev" {
						key := getenv("TYPESAFE_API_KEY")
						if key == "" {
							return errors.New("missing TYPESAFE_API_KEY for --pipeline jev")
						}
						base := envDefault(getenv, "TYPESAFE_BASE_URL", "https://api.typesafe.ai/v1")
						if err := ai.ValidateEndpoint(base); err != nil {
							return err
						}
						s.Matcher = ai.Jev{Key: key, Model: o.jevModel, BaseURL: base, HTTP: ai.NewTransport(), Observe: rec.observe}
					} else {
						s.Evaluator = st
					}
				}
			}

			var data []byte
			switch name {
			case "parse":
				v, e := s.Parse(ctx, args[0])
				err = e
				data = []byte(v.Text)
			case "extract":
				v, e := s.Extract(ctx, args[0])
				err = e
				if err == nil {
					data, err = json.MarshalIndent(v, "", "  ")
				}
			case "score":
				v, e := s.Score(ctx, args[0], jd, o.lang)
				err = e
				if err == nil {
					data, err = json.MarshalIndent(v, "", "  ")
				}
			}
			if err != nil {
				return err
			}
			if len(data) == 0 || data[len(data)-1] != '\n' {
				data = append(data, '\n')
			}
			if o.output != "" {
				if err = fileio.Write(o.output, data, o.force); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
			} else {
				_, err = out.Write(data)
			}
			if err == nil {
				logger.Info("completed", "command", name, "duration_ms", time.Since(start).Milliseconds())
			}
			return err
		}
		root.AddCommand(cmd)
	}
	return root
}
func envDefault(getenv func(string) string, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return fallback
}
func remote(o options, getenv func(string) string) (*ai.Remote, error) {
	var model, base string
	switch o.provider {
	case "gemini":
		model, base = "gemini-3.8-flash", "https://generativelanguage.googleapis.com/v1beta"
	case "deepseek":
		model, base = "deepseek-flash", "https://api.deepseek.com"
	case "openai":
		model, base = "gpt-6-astra", "https://api.openai.com/v1"
	case "kimi":
		model, base = "kimi-k3", "https://api.moonshot.ai/v1"
	default:
		return nil, errors.New("choose --provider gemini, deepseek, kimi or openai (or set RESUME_AI_PROVIDER)")
	}
	if o.model != "" {
		model = o.model
	}
	secret := getenv("RESUME_AI_API_KEY")
	if secret == "" {
		return nil, errors.New("missing RESUME_AI_API_KEY")
	}
	base = envDefault(getenv, "RESUME_AI_BASE_URL", base)
	if o.baseURL != "" {
		base = o.baseURL
	}
	if e := ai.ValidateEndpoint(base); e != nil {
		return nil, e
	}
	return &ai.Remote{Provider: o.provider, Model: model, Key: secret, BaseURL: base, HTTP: ai.NewTransport()}, nil
}
func checkPaths(inputs, outputs []string) error {
	seen := []string{}
	for _, dst := range outputs {
		if dst == "" {
			continue
		}
		for _, src := range append(inputs, seen...) {
			if src == "" {
				continue
			}
			a, e1 := filepath.Abs(src)
			b, e2 := filepath.Abs(dst)
			if e1 != nil || e2 != nil {
				return errors.New("invalid file path")
			}
			same := a == b
			i, e1 := os.Stat(src)
			j, e2 := os.Stat(dst)
			if e1 == nil && e2 == nil {
				same = same || os.SameFile(i, j)
			}
			if same {
				return errors.New("output paths must be distinct and must not overwrite inputs")
			}
		}
		seen = append(seen, dst)
	}
	return nil
}
