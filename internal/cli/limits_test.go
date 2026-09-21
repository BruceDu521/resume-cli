package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceLimitArguments(t *testing.T) {
	for _, flag := range []string{"--max-pdf-mib", "--max-text-kib", "--max-jd-kib"} {
		for _, value := range []string{"0", "-1", "9223372036854775807", "1.5"} {
			out, _, err := run("parse", "missing.pdf", flag, value)
			if err == nil || out != "" || strings.Contains(err.Error(), "文件不存在") {
				t.Fatal(flag, value, err)
			}
		}
	}
	for _, unit := range []int64{1 << 10, 1 << 20} {
		max := (int64(^uint(0)>>1) - 1) / unit
		if got, err := limitBytes(max, unit); err != nil || got != max*unit {
			t.Fatal(got, err)
		}
		if _, err := limitBytes(max+1, unit); err == nil {
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
