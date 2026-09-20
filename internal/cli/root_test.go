package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type noNetwork struct{}

func (noNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("NETWORK DISABLED IN TESTS")
}
func TestMain(m *testing.M) {
	http.DefaultTransport = noNetwork{}
	http.DefaultClient.Transport = noNetwork{}
	os.Exit(m.Run())
}
func run(args ...string) (string, string, error) {
	var out, logs bytes.Buffer
	cmd := New(&out, &logs, func(string) string { return "" })
	cmd.SetArgs(args)
	e := cmd.ExecuteContext(context.Background())
	return out.String(), logs.String(), e
}
func requirePDF(t *testing.T) {
	t.Helper()
	if _, e := exec.LookPath("pdftotext"); e != nil {
		t.Skip("Poppler unavailable")
	}
}
func TestOfflineEndToEnd(t *testing.T) {
	requirePDF(t)
	for _, lang := range []string{"zh", "en"} {
		out, logs, e := run("score", "../../testdata/resume-zh.pdf", "--jd", "../../testdata/jd.txt", "--mock", "--lang", lang)
		if e != nil {
			t.Fatal(e)
		}
		var v map[string]any
		if e = json.Unmarshal([]byte(out), &v); e != nil {
			t.Fatal(e, out)
		}
		if v["language"] != lang || v["mock"] != true || v["overall_score"] != 83.0 || !strings.Contains(logs, "MOCK") {
			t.Fatal(out, logs)
		}
	}
	out, _, e := run("extract", "../../testdata/resume-en.pdf", "--mock")
	if e != nil || !strings.Contains(out, "Lin Yuan") {
		t.Fatal(out, e)
	}
	out, _, e = run("parse", "../../testdata/resume-zh.pdf")
	if e != nil || !strings.Contains(out, "林予安") {
		t.Fatal(out, e)
	}
}
func TestFilesAndCache(t *testing.T) {
	requirePDF(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "out.json")
	stats := filepath.Join(dir, "stats.json")
	args := []string{"score", "../../testdata/resume-en.pdf", "--jd", "../../testdata/jd-en.txt", "--mock", "--output", out, "--stats", stats, "--cache-dir", filepath.Join(dir, "cache")}
	stdout, _, e := run(args...)
	if e != nil || stdout != "" {
		t.Fatal(stdout, e)
	}
	b, _ := os.ReadFile(out)
	if !json.Valid(b) {
		t.Fatal(string(b))
	}
	b, _ = os.ReadFile(stats)
	if !strings.Contains(string(b), `"calls": []`) {
		t.Fatal("mock called provider", string(b))
	}
	if _, _, e = run(args...); e == nil {
		t.Fatal("clobbered")
	}
	_, logs, e := run(append(args, "--force")...)
	if e != nil || !strings.Contains(logs, "cache hit") {
		t.Fatal(logs, e)
	}
}
func TestValidation(t *testing.T) {
	for _, args := range [][]string{{"score", "x"}, {"extract", "x"}, {"extract", "x", "--provider", "gemini"}, {"parse", "x", "--lang", "fr"}, {"parse", "x", "--timeout", "0s"}, {"parse", "x", "--pipeline", "other"}, {"parse", "x", "--report", "other"}, {"parse", "x", "--output", "x", "--force"}, {"score", "x", "--jd", "x", "--mock", "--output", "out", "--stats", "out"}, {"extract", "x", "--mock", "--report", "ai"}, {"score", "x", "--jd", "missing"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if out, _, e := run(args...); e == nil || out != "" {
				t.Fatal(out, e)
			}
		})
	}
}
func TestHelpAndPrivateInputs(t *testing.T) {
	out, _, e := run("--help")
	if e != nil || !strings.Contains(out, "score") {
		t.Fatal(out, e)
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "link")
	os.WriteFile(src, []byte("original"), 0600)
	if e = os.Symlink(src, dst); e != nil {
		t.Skip(e)
	}
	if e = checkPaths([]string{src}, []string{dst}); e == nil {
		t.Fatal("input alias overwrite")
	}
}
func TestRemoteConfig(t *testing.T) {
	for _, p := range []string{"gemini", "deepseek", "openai", "kimi"} {
		r, e := remote(options{provider: p}, func(k string) string {
			if strings.HasSuffix(k, "API_KEY") {
				return "synthetic"
			}
			return ""
		})
		if e != nil || r.Key != "synthetic" || r.Provider != p {
			t.Fatal(r, e)
		}
	}
	if _, e := remote(options{provider: "gemini"}, func(k string) string {
		if k == "RESUME_AI_BASE_URL" {
			return "http://insecure"
		}
		return "synthetic"
	}); e == nil {
		t.Fatal("insecure endpoint")
	}
}

func TestFailureStats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	_, _, err := run("extract", "missing.pdf", "--stats", path)
	if err == nil {
		t.Fatal("expected configuration error")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Success bool
		Calls   []any
	}
	if err = json.Unmarshal(data, &v); err != nil || v.Success || len(v.Calls) != 0 {
		t.Fatal(string(data), err)
	}
}
