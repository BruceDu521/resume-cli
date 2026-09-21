package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	samples "resume-cli"
	"strings"
	"testing"
)

func TestExportSamples(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if _, _, err := run("samples", dir); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 4 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	for _, entry := range entries {
		got, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		want, err := samples.Files.ReadFile("testdata/" + entry.Name())
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("fixture %s changed: %v", entry.Name(), err)
		}
	}
	sentinel := filepath.Join(dir, "jd.txt")
	if err := os.WriteFile(sentinel, []byte("keep my changes"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run("samples", dir, "--force"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("existing directory must be preserved: %v", err)
	}
	got, _ := os.ReadFile(sentinel)
	if string(got) != "keep my changes" {
		t.Fatal("overwrote existing sample")
	}
	if _, _, err := run("samples"); err == nil {
		t.Fatal("accepted missing destination")
	}
	if _, _, err := run("samples", filepath.Join(t.TempDir(), "missing", "demo")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing parent: %v", err)
	}
}

func TestExportedSamplesRunOffline(t *testing.T) {
	requirePDF(t)
	dir := filepath.Join(t.TempDir(), "demo")
	if _, _, err := run("samples", dir); err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"zh", "en"} {
		resume := filepath.Join(dir, "resume-"+lang+".pdf")
		jd := filepath.Join(dir, "jd.txt")
		if lang == "en" {
			jd = filepath.Join(dir, "jd-en.txt")
		}
		if out, _, err := run("parse", resume); err != nil || !strings.Contains(out, "RESUME_CLI_DEMO_V1") {
			t.Fatal(out, err)
		}
		if out, _, err := run("extract", resume, "--mock"); err != nil || !strings.Contains(out, "lin.yuan@example.com") {
			t.Fatal(out, err)
		}
		if out, _, err := run("score", resume, "--jd", jd, "--mock", "--lang", lang); err != nil || !strings.Contains(out, `"mock": true`) {
			t.Fatal(out, err)
		}
	}
}
