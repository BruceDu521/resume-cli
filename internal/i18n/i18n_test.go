package i18n

import (
	"errors"
	"strings"
	"testing"
)

func TestLocalePrecedence(t *testing.T) {
	for _, tt := range []struct {
		env  map[string]string
		want string
	}{
		{nil, "en"},
		{map[string]string{"LANG": "zh_CN.UTF-8"}, "zh"},
		{map[string]string{"LANG": "zh-TW"}, "zh"},
		{map[string]string{"LANG": "en_US.UTF-8"}, "en"},
		{map[string]string{"LANG": "zh_CN", "LC_MESSAGES": "en_GB"}, "en"},
		{map[string]string{"LANG": "en_US", "LC_MESSAGES": "en_US", "LC_ALL": "zh_HK.UTF-8"}, "zh"},
		{map[string]string{"LC_ALL": "C.UTF-8", "LANG": "zh_CN"}, "en"},
		{map[string]string{"LC_ALL": "POSIX", "LC_MESSAGES": "zh_CN"}, "en"},
		{map[string]string{"LC_ALL": "zh_CN", "RESUME_CLI_LANG": "en"}, "en"},
		{map[string]string{"LC_ALL": "en_US", "RESUME_CLI_LANG": "zh"}, "zh"},
		{map[string]string{"LANG": "de_DE.UTF-8"}, "en"},
		{map[string]string{"LC_MESSAGES": " ", "LANG": "ZH_CN.UTF-8"}, "zh"},
	} {
		if got := Detect(func(k string) string { return tt.env[k] }); got != tt.want {
			t.Fatalf("%v: %s", tt.env, got)
		}
	}
}
func TestErrorsKeepCausesAndArguments(t *testing.T) {
	cause := errors.New("untranslated diagnostic")
	e := Errorf("简历 PDF：%w", cause)
	if !errors.Is(e, cause) {
		t.Fatal("lost cause")
	}
	if got := Render("en", e); got != "Resume PDF: untranslated diagnostic" {
		t.Fatal(got)
	}
	joined := errors.Join(e, New("文件路径无效，请检查路径写法"))
	if got := Render("en", joined); !strings.Contains(got, "Invalid file path") || !strings.Contains(got, "Resume PDF") {
		t.Fatal(got)
	}
	if Render("en", nil) != "" {
		t.Fatal("nil error")
	}
	// A translated format string must not translate arbitrary arguments.
	e = Errorf("请提供一个 PDF 简历路径，例如 %s resume.pdf；查看完整用法：%s --help", "中文命令", "中文命令")
	if !strings.Contains(Render("en", e), "中文命令") {
		t.Fatal("argument changed")
	}
}
func TestCatalogHasNonemptyTranslations(t *testing.T) {
	for key, value := range english {
		if key == "" || value == "" || key == value {
			t.Fatal(key, value)
		}
	}
}
