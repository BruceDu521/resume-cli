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
		{"text limit", "head -c 170000 /dev/zero", "131072"},
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

func TestConfiguredInputLimits(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-parser")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nprintf '12345678'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "resume.pdf")
	content := []byte("%PDF-test")
	if err := os.WriteFile(source, content, 0600); err != nil {
		t.Fatal(err)
	}
	p := Parser{Binary: bin, MaxPDFBytes: int64(len(content)), MaxTextBytes: 8}
	if d, err := p.Parse(context.Background(), source); err != nil || d.Text != "12345678" {
		t.Fatal(d, err)
	}
	p.MaxPDFBytes--
	if _, err := p.Parse(context.Background(), source); err == nil || !strings.Contains(err.Error(), "文件过大") {
		t.Fatal(err)
	}
	p.MaxPDFBytes++
	p.MaxTextBytes = 7
	if _, err := p.Parse(context.Background(), source); err == nil || !strings.Contains(err.Error(), "7 字节") {
		t.Fatal(err)
	}
	p.MaxTextBytes = 16
	if d, err := p.Parse(context.Background(), source); err != nil || d.Text != "12345678" {
		t.Fatal(d, err)
	}
}
