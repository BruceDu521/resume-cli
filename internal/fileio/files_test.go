package fileio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "out")
	if e := Write(p, []byte("old"), false); e != nil {
		t.Fatal(e)
	}
	if e := Write(p, []byte("new"), false); e == nil {
		t.Fatal("overwrote existing file")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "old" {
		t.Fatal("clobbered")
	}
	if e := Write(p, []byte("new"), true); e != nil {
		t.Fatal(e)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0600 {
		t.Fatal("private file permissions")
	}
	if _, e := Read(p, 2); e == nil {
		t.Fatal("limit")
	}
	if _, e := Read(dir, 100); e == nil {
		t.Fatal("directory")
	}
	if _, e := Read(filepath.Join(dir, "missing"), 100); e == nil {
		t.Fatal("missing")
	}
}
func TestText(t *testing.T) {
	for _, b := range [][]byte{{0xff}, []byte(" \n")} {
		p := filepath.Join(t.TempDir(), "x")
		os.WriteFile(p, b, 0600)
		if _, e := Text(p, 100); e == nil {
			t.Fatal("accepted invalid text")
		}
	}
	p := filepath.Join(t.TempDir(), "x")
	os.WriteFile(p, []byte("\ufeff你好\n"), 0600)
	s, e := Text(p, 100)
	if e != nil || s != "你好" {
		t.Fatal(s, e)
	}
}

func TestUTF8ByteLimitBoundary(t *testing.T) {
	p := filepath.Join(t.TempDir(), "jd.txt")
	// Exactly 64 KiB with multibyte Chinese text: count bytes, not characters.
	data := []byte(strings.Repeat("中", (64<<10)/3) + "x")
	if len(data) != 64<<10 {
		t.Fatal(len(data))
	}
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Text(p, 64<<10); err != nil {
		t.Fatal("boundary rejected", err)
	}
	if err := os.WriteFile(p, append(data, 'x'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Text(p, 64<<10); err == nil {
		t.Fatal("over-limit input accepted")
	}
}
