package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFriendlyInputErrorsBeforeCredentials(t *testing.T) {
	requirePDF(t)
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if e := os.WriteFile(p, data, 0600); e != nil {
			t.Fatal(e)
		}
		return p
	}
	empty := write("empty.pdf", nil)
	whitespace := write("blank.txt", []byte("\ufeff \n\t"))
	nonPDF := write("fake.pdf", []byte("plain text"))
	broken := write("broken.pdf", []byte("%PDF-broken"))
	invalid := write("encoding.txt", []byte{0xff, 0xfe})
	bigJD := write("big.txt", bytes.Repeat([]byte("a"), (64<<10)+1))
	bigPDF := write("big.pdf", nil)
	if err := os.Truncate(bigPDF, (20<<20)+1); err != nil {
		t.Fatal(err)
	}
	resume := "../../testdata/resume-en.pdf"
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"missing resume", []string{"extract", filepath.Join(dir, "missing.pdf")}, "简历 PDF："},
		{"empty resume", []string{"parse", empty}, "文件为空"},
		{"not PDF", []string{"extract", nonPDF}, "文件不是 PDF"},
		{"broken PDF", []string{"parse", broken}, "PDF 无法解析"},
		{"no PDF text", []string{"parse", "../../testdata/empty.pdf"}, "OCR"},
		{"PDF directory", []string{"parse", dir}, "不是普通文件"},
		{"large PDF", []string{"parse", bigPDF}, "文件过大"},
		{"missing JD", []string{"score", resume, "--jd", filepath.Join(dir, "missing.txt")}, "岗位描述（JD）："},
		{"empty JD", []string{"score", resume, "--jd", empty}, "文件为空"},
		{"blank JD", []string{"score", resume, "--jd", whitespace}, "仅含空白"},
		{"invalid encoding", []string{"score", resume, "--jd", invalid}, "UTF-8"},
		{"PDF as JD", []string{"score", resume, "--jd", resume}, "需要纯文本岗位描述"},
		{"JD directory", []string{"score", resume, "--jd", dir}, "不是普通文件"},
		{"large JD", []string{"score", resume, "--jd", bigJD}, "文件过大"},
		{"missing argument", []string{"extract"}, "请提供一个 PDF"},
		{"missing JD flag", []string{"score", resume}, "缺少岗位描述路径"},
		{"output directory missing", []string{"parse", resume, "--output", filepath.Join(dir, "missing", "out.txt")}, "输出目录不存在"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := New(&stdout, &stderr, func(key string) string {
				if key == "RESUME_AI_API_KEY" {
					t.Fatal("input error must precede credential lookup")
				}
				return ""
			})
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.want) || stdout.Len() != 0 {
				t.Fatalf("%v; stdout=%s", err, stdout.String())
			}
			for _, raw := range []string{"stat ", "open ", "exit status", "Syntax Error", "resume-cli-"} {
				if strings.Contains(err.Error(), raw) {
					t.Fatal("raw error leaked", err)
				}
			}
		})
	}
}

func TestHelpExplainsCommandsAndEnvironment(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"score", "--help"}, {"extract", "--help"}, {"help", "score"}} {
		out, _, err := run(args...)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"示例：", "RESUME_AI_PROVIDER", "RESUME_AI_API_KEY", "RESUME_AI_MODEL", "不会自动读取 .env", "90s", "usage.json"} {
			if !strings.Contains(out, want) {
				t.Fatal("help missing", want)
			}
		}
		if strings.Contains(out, "completion") {
			t.Fatal("unexpected completion command")
		}
	}
	out, _, _ := run("--help")
	for _, want := range []string{"从本地 PDF 提取纯文本", "调用 AI 提取姓名", "生成匹配评分"} {
		if !strings.Contains(out, want) {
			t.Fatal("missing command description", want)
		}
	}
}
