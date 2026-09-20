package pdf

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserFixtures(t *testing.T) {
	if _, e := exec.LookPath("pdftotext"); e != nil {
		t.Skip("install Poppler to run local PDF integration cases")
	}
	for _, tt := range []struct{ name, want string }{{"resume-en.pdf", "Lin Yuan"}, {"resume-zh.pdf", "林予安"}, {"columns.pdf", "Right column content"}} {
		t.Run(tt.name, func(t *testing.T) {
			d, e := (Parser{}).Parse(context.Background(), "../../testdata/"+tt.name)
			if e != nil || !strings.Contains(d.Text, tt.want) || len(d.Blocks) == 0 {
				t.Fatal(d, e)
			}
		})
	}
	if _, e := (Parser{}).Parse(context.Background(), "../../testdata/empty.pdf"); e == nil {
		t.Fatal("empty PDF")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := (Parser{}).Parse(ctx, "../../testdata/resume-en.pdf"); e == nil {
		t.Fatal("cancel ignored")
	}
}
func TestInvalidFiles(t *testing.T) {
	p := filepath.Join(t.TempDir(), "fake.pdf")
	os.WriteFile(p, []byte("not pdf"), 0600)
	if _, e := (Parser{}).Parse(context.Background(), p); e == nil {
		t.Fatal("non-PDF")
	}
	os.WriteFile(p, []byte("%PDF-broken"), 0600)
	if _, e := (Parser{}).Parse(context.Background(), p); e == nil {
		t.Fatal("broken PDF")
	}
	if _, e := (Parser{Binary: "/not/a/real/program"}).Parse(context.Background(), p); e == nil {
		t.Fatal("missing dependency")
	}
	f, _ := os.Create(p)
	f.Truncate(MaxPDFBytes + 1)
	f.Close()
	if _, e := (Parser{}).Parse(context.Background(), p); e == nil {
		t.Fatal("oversized PDF")
	}
}
func TestBounded(t *testing.T) {
	b := bounded{limit: 2}
	if _, e := b.Write([]byte("xx")); e != nil {
		t.Fatal(e)
	}
	if _, e := b.Write([]byte("x")); e == nil {
		t.Fatal("unbounded")
	}
}
