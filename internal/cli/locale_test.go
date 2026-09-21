package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"resume-cli/internal/i18n"
)

func executeLocale(locale string, args ...string) (string, error) {
	var out, logs bytes.Buffer
	cmd := New(&out, &logs, func(k string) string {
		if k == "LANG" {
			return locale
		}
		return ""
	})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}
func TestInterfaceLanguageIndependentOfReport(t *testing.T) {
	requirePDF(t)
	for _, locale := range []string{"zh_CN.UTF-8", "en_US.UTF-8"} {
		for _, reportLang := range []string{"", "en"} {
			args := []string{"score", "../../testdata/resume-zh.pdf", "--jd", "../../testdata/jd.txt", "--mock"}
			if reportLang != "" {
				args = append(args, "--lang", reportLang)
			}
			out, err := executeLocale(locale, args...)
			if err != nil {
				t.Fatal(err)
			}
			var result struct{ Language, Comment string }
			if err = json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatal(err)
			}
			want := "zh"
			if reportLang != "" {
				want = reportLang
			}
			if result.Language != want {
				t.Fatalf("locale=%s report=%s", locale, result.Language)
			}
			if want == "zh" && !strings.Contains(result.Comment, "合成演示") {
				t.Fatal(result.Comment)
			}
			if want == "en" && !strings.Contains(result.Comment, "Synthetic demo") {
				t.Fatal(result.Comment)
			}
		}
	}
}
func TestEnglishHelpAndErrorRendering(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"score", "--help"}, {"help", "extract"}} {
		out, err := executeLocale("en_US.UTF-8", args...)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"Usage:", "Examples:", "RESUME_CLI_LANG", "LC_ALL", "RESUME_AI_PROVIDER", "Chinese by default", "32 MiB", "64 KiB", "200 MiB"} {
			if !strings.Contains(out, want) {
				t.Fatal("missing translation", want)
			}
		}
		for _, untranslated := range []string{"用法", "查看", "简历", "岗位", "必填", "统计", "密钥"} {
			if strings.Contains(out, untranslated) {
				t.Fatal("untranslated help", untranslated)
			}
		}
	}
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"extract", "missing.pdf"}, "Resume PDF: \"missing.pdf\": File not found"},
		{[]string{"score", "missing.pdf", "--jd", "missing.txt"}, "Job description (JD):"},
		{[]string{"extract"}, "Provide one PDF resume path"},
		{[]string{"score", "missing.pdf"}, "Missing job description path"},
		{[]string{"parse", "x", "--timeout", "0s"}, "--timeout must be greater than zero"},
		{[]string{"parse", "x", "--bogus"}, "Unknown or invalid option"},
	} {
		_, err := executeLocale("en_US.UTF-8", tt.args...)
		if err == nil || !strings.Contains(i18n.Render("en", err), tt.want) {
			t.Fatal(err, tt.want)
		}
	}
}
