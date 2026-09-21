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
