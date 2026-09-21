package pdf

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserFailuresHideProcessDetails(t *testing.T) {
	for _, tt := range []struct{ name, script, want string }{
		{"encrypted", "printf 'Incorrect password private-provider-detail' >&2; exit 1", "已加密"},
		{"broken", "printf 'Syntax Error private-provider-detail' >&2; exit 1", "无法解析"},
		{"empty", "exit 0", "没有可提取的文字"},
		{"mapping", "printf 'Missing language pack' >&2; printf 'text'", "poppler-data"},
		{"text limit", "head -c 170000 /dev/zero", "160 KiB"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "fake-pdftotext")
			if e := os.WriteFile(bin, []byte("#!/bin/sh\n"+tt.script+"\n"), 0700); e != nil {
				t.Fatal(e)
			}
			_, err := (Parser{Binary: bin}).Parse(context.Background(), "../../testdata/resume-en.pdf")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatal(err)
			}
			if strings.Contains(err.Error(), "private-provider-detail") || strings.Contains(err.Error(), "exit status") || strings.Contains(err.Error(), bin) {
				t.Fatal("raw process detail leaked", err)
			}
		})
	}
	_, err := (Parser{Binary: "/nonexistent/pdftotext"}).Parse(context.Background(), "../../testdata/resume-en.pdf")
	if err == nil || !strings.Contains(err.Error(), "brew install poppler") {
		t.Fatal(err)
	}
}
