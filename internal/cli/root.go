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
	"resume-cli/internal/domain"
	"strings"
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
	output, jd, lang, provider, model, baseURL, cacheDir, stats string
	mock, force                                                 bool
	timeout                                                     time.Duration
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
func (r *recorder) snapshot(elapsed time.Duration, mock bool, success bool) any {
	r.mu.Lock()
	defer r.mu.Unlock()
	return struct {
		Success bool       `json:"success"`
		Elapsed int64      `json:"elapsed_ms"`
		Mock    bool       `json:"mock"`
		Calls   []ai.Usage `json:"calls"`
		Hits    []string   `json:"cache_hits"`
	}{success, elapsed.Milliseconds(), mock, append([]ai.Usage{}, r.calls...), append([]string{}, r.hits...)}
}
func New(out, errOut io.Writer, getenv func(string) string) *cobra.Command {
	if getenv == nil {
		getenv = os.Getenv
	}
	o := options{}
	root := &cobra.Command{Use: "resume-cli", Short: "本地 PDF 简历解析与 AI 岗位匹配", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(errOut)
	configureHelp(root)
	root.Example = "  resume-cli parse resume.pdf\n  resume-cli extract resume.pdf\n  resume-cli score resume.pdf --jd jd.txt\n  resume-cli score testdata/resume-zh.pdf --jd testdata/jd.txt --mock"
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("参数无法识别或格式不正确：%v。请运行 %s --help 查看用法。", err, cmd.CommandPath())
	})
	f := root.PersistentFlags()
	f.StringVar(&o.output, "output", "", "结果保存路径，如 result.json；不填则打印到终端")
	f.BoolVar(&o.force, "force", false, "允许覆盖 --output / --stats 指定的已有文件；不会覆盖输入")
	f.BoolVar(&o.mock, "mock", false, "不用 API key 演示；仅支持 testdata 中的合成简历和 JD")
	f.StringVar(&o.lang, "lang", "zh", "score 评语和面试问题的语言：zh 中文，en 英文")
	f.StringVar(&o.provider, "provider", getenv("RESUME_AI_PROVIDER"), "AI 厂商（也可设 RESUME_AI_PROVIDER）；可选值见下方配置")
	f.StringVar(&o.model, "model", getenv("RESUME_AI_MODEL"), "模型型号，如 deepseek-flash（也可设 RESUME_AI_MODEL）")
	f.StringVar(&o.baseURL, "base-url", getenv("RESUME_AI_BASE_URL"), "自定义 HTTPS API 地址（RESUME_AI_BASE_URL）；通常不用填")
	f.StringVar(&o.cacheDir, "cache-dir", "", "仅 extract：缓存目录，如 .cache；24 小时内复用提取结果")
	f.StringVar(&o.stats, "stats", "", "统计文件路径，如 usage.json；记录耗时、token 和估算费用")
	f.DurationVar(&o.timeout, "timeout", 90*time.Second, "最多等待多久，如 90s 或 2m（上限 10m）")
	for _, name := range []string{"parse", "extract", "score"} {
		name := name
		cmd := &cobra.Command{Use: name + " <简历.pdf>", Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("请提供一个 PDF 简历路径，例如 %s resume.pdf；查看完整用法：%s --help", cmd.CommandPath(), cmd.CommandPath())
			}
			if name == "score" && strings.TrimSpace(o.jd) == "" {
				return errors.New("缺少岗位描述路径，请使用 --jd jd.txt 指定 UTF-8 文本文件")
			}
			return nil
		}}
		switch name {
		case "parse":
			cmd.Short = "从本地 PDF 提取纯文本，不调用 AI"
			cmd.Example = "  resume-cli parse resume.pdf\n  resume-cli parse resume.pdf --output resume.txt"
		case "extract":
			cmd.Short = "调用 AI 提取姓名、联系方式、教育经历和技能，输出 JSON"
			cmd.Example = "  resume-cli extract resume.pdf\n  resume-cli extract resume.pdf --provider deepseek --output extracted.json\n  resume-cli extract testdata/resume-zh.pdf --mock"
		case "score":
			cmd.Short = "结合简历与岗位描述，生成匹配评分、评语和面试问题"
			cmd.Example = "  resume-cli score resume.pdf --jd jd.txt\n  resume-cli score resume.pdf --jd jd.txt --lang en --output result.json\n  resume-cli score testdata/resume-zh.pdf --jd testdata/jd.txt --mock"
		}
		if name == "score" {
			cmd.Flags().StringVar(&o.jd, "jd", "", "必填：UTF-8 岗位描述文件路径，如 jd.txt（不是 PDF）")
		}
		cmd.RunE = func(cmd *cobra.Command, args []string) (runErr error) {
			start := time.Now()
			if o.timeout <= 0 || o.timeout > 10*time.Minute {
				return errors.New("--timeout 必须大于 0 且不超过 10m，例如 90s 或 2m")
			}
			if o.lang != "zh" && o.lang != "en" {
				return errors.New("--lang 仅支持 zh（中文）或 en（英文）")
			}
			if name != "extract" && o.cacheDir != "" {
				return errors.New("--cache-dir 仅适用于 extract 信息提取命令")
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
				return errors.New("RESUME_LOG_LEVEL 仅支持 debug、info、warn 或 error")
			}
			logger := slog.New(slog.NewTextHandler(errOut, &slog.HandlerOptions{Level: level}))
			rec := &recorder{logger: logger}
			for _, path := range []string{o.output, o.stats} {
				if path == "" {
					continue
				}
				if _, e := os.Lstat(path); e == nil && !o.force {
					return fileio.WriteError(path, os.ErrExist)
				} else if e != nil && !errors.Is(e, os.ErrNotExist) {
					return fileio.WriteError(path, e)
				}
			}
			// Preserve timing and observed usage on failures as well as successes.
			// Do not include source text or untrusted provider errors in this file.
			defer func() {
				if o.stats == "" {
					return
				}
				stats, e := json.MarshalIndent(rec.snapshot(time.Since(start), o.mock, runErr == nil), "", "  ")
				if e == nil {
					e = fileio.Write(o.stats, append(stats, '\n'), o.force)
				}
				if e != nil {
					runErr = errors.Join(runErr, fmt.Errorf("统计文件保存失败：%w", e))
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
					return fmt.Errorf("岗位描述（JD）：%w", err)
				}
			}
			// Validate local inputs before configuring or calling an AI provider.
			doc, err := s.Parse(ctx, args[0])
			if err != nil {
				return err
			}
			s.Parser = parsedInput{doc}

			if name != "parse" {
				if o.mock {
					logger.Warn("MOCK: synthetic fixture demonstration; no AI requests")
					m := ai.Mock{}
					s.Extractor = m
					s.Evaluator = m
					s.Identity = "mock-v1"
				} else {
					g, err := remote(o, getenv)
					if err != nil {
						return err
					}
					st := ai.Structurer{Generator: g, Observe: rec.observe}
					s.Extractor = st
					s.Identity = g.Identity()
					s.Evaluator = st
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
					return fmt.Errorf("结果保存失败：%w", err)
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
	case "anthropic", "claude":
		o.provider = "anthropic"
		model, base = "claude-sonnet-5", "https://api.anthropic.com/v1"
	case "kimi":
		model, base = "kimi-k3", "https://api.moonshot.ai/v1"
	default:
		return nil, errors.New("请选择 AI 厂商：设置 RESUME_AI_PROVIDER 或传入 --provider；支持 deepseek、gemini、kimi、openai、anthropic（或 claude）")
	}
	if o.model != "" {
		model = o.model
	}
	secret := getenv("RESUME_AI_API_KEY")
	if secret == "" {
		return nil, errors.New("未配置 RESUME_AI_API_KEY，请设置所选厂商的 API key；离线演示可使用 --mock")
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
				return errors.New("文件路径无效，请检查路径写法")
			}
			same := a == b
			i, e1 := os.Stat(src)
			j, e2 := os.Stat(dst)
			if e1 == nil && e2 == nil {
				same = same || os.SameFile(i, j)
			}
			if same {
				return errors.New("输出路径不能覆盖输入文件；--output 与 --stats 也不能指向同一个文件")
			}
		}
		seen = append(seen, dst)
	}
	return nil
}

// parsedInput avoids running Poppler twice after input validation.
type parsedInput struct{ document domain.Document }

func (p parsedInput) Parse(ctx context.Context, _ string) (domain.Document, error) {
	return p.document, ctx.Err()
}
