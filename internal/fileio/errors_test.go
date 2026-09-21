package fileio

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestFriendlyErrorsPreserveCauses(t *testing.T) {
	for _, tt := range []struct {
		cause       error
		read, write string
	}{
		{os.ErrNotExist, "文件不存在", "输出目录不存在"},
		{os.ErrPermission, "没有读取权限", "没有写入权限"},
		{os.ErrExist, "无法读取文件", "文件已存在"},
		{errors.New("private OS detail"), "无法读取文件", "无法写入文件"},
	} {
		cause := &os.PathError{Op: "stat", Path: "private-temp-path", Err: tt.cause}
		for _, tc := range []struct {
			err  error
			want string
		}{{ReadError("user.txt", cause), tt.read}, {WriteError("user.txt", cause), tt.write}} {
			if !errors.Is(tc.err, tt.cause) || !strings.Contains(tc.err.Error(), tc.want) || !strings.Contains(tc.err.Error(), "user.txt") {
				t.Fatal(tc.err)
			}
			for _, raw := range []string{"stat ", "private-temp-path", "private OS detail"} {
				if strings.Contains(tc.err.Error(), raw) {
					t.Fatal("raw OS error leaked", tc.err)
				}
			}
		}
	}
}

func TestEnglishFileErrorsPreserveFilename(t *testing.T) {
	path := "文件不存在，请检查路径和文件名。.pdf"
	err := ReadError(path, os.ErrNotExist)
	e := err.(*Error)
	got := e.Localize("en")
	if !strings.Contains(got, path) || !strings.Contains(got, "File not found") {
		t.Fatal(got)
	}
	e = &Error{Path: "large.pdf", Message: "文件过大（上限 %d 字节），请缩小文件后重试。", Args: []any{64 << 10}}
	if got = e.Localize("en"); !strings.Contains(got, "65536 bytes") {
		t.Fatal(got)
	}
}
