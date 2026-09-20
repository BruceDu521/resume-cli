package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLifecycle(t *testing.T) {
	now := time.Now()
	s := Store{Dir: t.TempDir(), TTL: time.Hour, Now: func() time.Time { return now }}
	var got map[string]string
	hit, e := s.Get("key", &got)
	if hit || e != nil {
		t.Fatal(hit, e)
	}
	if e = s.Put("key", map[string]string{"name": "synthetic"}); e != nil {
		t.Fatal(e)
	}
	hit, e = s.Get("key", &got)
	if !hit || e != nil || got["name"] != "synthetic" {
		t.Fatal(hit, e)
	}
	now = now.Add(2 * time.Hour)
	hit, e = s.Get("key", &got)
	if hit || e != nil {
		t.Fatal("expired hit")
	}
	now = now.Add(-3 * time.Hour)
	hit, _ = s.Get("key", &got)
	if hit {
		t.Fatal("future entry")
	}
	os.WriteFile(s.path("key"), []byte("broken"), 0600)
	hit, e = s.Get("key", &got)
	if hit || e != nil {
		t.Fatal("corrupt entry")
	}
}
func TestDisabledAndSymlink(t *testing.T) {
	s := Store{}
	if e := s.Put("x", nil); e != nil {
		t.Fatal(e)
	}
	var out any
	if hit, e := s.Get("x", &out); hit || e != nil {
		t.Fatal(hit, e)
	}
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	if e := os.Symlink(t.TempDir(), link); e != nil {
		t.Skip(e)
	}
	s.Dir = link
	if s.Put("x", 1) == nil {
		t.Fatal("symlink root accepted")
	}
}
