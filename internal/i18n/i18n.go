// Package i18n localizes terminal messages independently of AI report language.
package i18n

import (
	"fmt"
	"strings"
)

// Detect follows POSIX locale precedence. Unsupported/unset locales use English.
// RESUME_CLI_LANG overrides the environment, including when it is C/POSIX.
func Detect(getenv func(string) string) string {
	for _, key := range []string{"RESUME_CLI_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(strings.TrimSpace(getenv(key)))
		if v == "" {
			continue
		}
		base := strings.FieldsFunc(v, func(r rune) bool { return r == '_' || r == '-' || r == '.' || r == '@' })
		if len(base) > 0 && base[0] == "zh" {
			return "zh"
		}
		return "en"
	}
	return "en"
}

// Source strings are message keys (gettext style). Only application-owned
// messages are translated, never filenames, model output or arbitrary substrings.
func Text(lang, source string) string {
	if lang == "en" {
		if translated, ok := english[source]; ok {
			return translated
		}
	}
	return source
}

type message struct {
	source string
	args   []any
}

func New(source string) error                 { return &message{source: source} }
func Errorf(source string, args ...any) error { return &message{source: source, args: args} }
func (m *message) Error() string              { return m.Localize("zh") }
func (m *message) Localize(lang string) string {
	args := append([]any(nil), m.args...)
	for i, arg := range args {
		if err, ok := arg.(error); ok {
			args[i] = Render(lang, err)
		}
	}
	return fmt.Sprintf(strings.ReplaceAll(Text(lang, m.source), "%w", "%v"), args...)
}
func (m *message) Unwrap() []error {
	var errs []error
	for _, arg := range m.args {
		if e, ok := arg.(error); ok {
			errs = append(errs, e)
		}
	}
	return errs
}
func Render(lang string, err error) string {
	if err == nil {
		return ""
	}
	if localized, ok := err.(interface{ Localize(string) string }); ok {
		return localized.Localize(lang)
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var parts []string
		for _, cause := range joined.Unwrap() {
			parts = append(parts, Render(lang, cause))
		}
		return strings.Join(parts, "\n")
	}
	return err.Error()
}
