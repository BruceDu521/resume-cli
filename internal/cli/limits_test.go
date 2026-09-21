package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestResourceLimitArguments(t *testing.T) {
	for _, flag := range []string{"--max-pdf-mib", "--max-text-kib", "--max-jd-kib"} {
		for _, value := range []string{"0", "-1", "9223372036854775807", "1.5", "abc", "9223372036854775808"} {
			out, _, err := run("parse", "missing.pdf", flag, value)
			if err == nil || out != "" || strings.Contains(err.Error(), "文件不存在") {
				t.Fatal(flag, value, err)
			}
		}
	}

	for _, tt := range []struct {
		flag string
		max  int64
	}{{"--max-pdf-mib", 200}, {"--max-text-kib", 256}, {"--max-jd-kib", 128}} {
		for _, value := range []int64{1, tt.max} {
			_, _, err := run("parse", "missing.pdf", tt.flag, strconv.FormatInt(value, 10))
			if err == nil || !strings.Contains(err.Error(), "文件不存在") {
				t.Fatal("valid value rejected", tt.flag, value, err)
			}
		}
		_, _, err := run("parse", "missing.pdf", tt.flag, strconv.FormatInt(tt.max+1, 10))
		if err == nil || !strings.Contains(err.Error(), strconv.FormatInt(tt.max, 10)) {
			t.Fatal("ceiling not enforced", tt.flag, err)
		}
	}
	for _, unit := range []int64{1 << 10, 1 << 20} {
		max := (int64(^uint(0)>>1) - 1) / unit
		if _, err := limitBytes(max+1, unit, max+1); err == nil {
			t.Fatal("overflow accepted")
		}
	}

}
func TestJDLimitOverride(t *testing.T) {
	requirePDF(t)
	path := filepath.Join(t.TempDir(), "jd.txt")
	data := "RESUME_CLI_JD_V1" + strings.Repeat("a", 1100) + "\nGo / PostgreSQL\nKubernetes\n本科"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"score", "../../testdata/resume-zh.pdf", "--jd", path, "--mock"}
	if _, _, err := run(append(args, "--max-jd-kib", "1")...); err == nil || !strings.Contains(err.Error(), "文件过大") {
		t.Fatal(err)
	}
	if _, _, err := run(append(args, "--max-jd-kib", "2")...); err != nil {
		t.Fatal(err)
	}
}

func TestResourceDefaults(t *testing.T) {
	cmd := New(&bytes.Buffer{}, &bytes.Buffer{}, func(string) string { return "" })
	for flag, want := range map[string]int64{"max-pdf-mib": 32, "max-text-kib": 64, "max-jd-kib": 32} {
		got, err := cmd.PersistentFlags().GetInt64(flag)
		if err != nil || got != want {
			t.Fatal(flag, got, err)
		}
	}
}
