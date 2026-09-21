package fileio

import (
	"errors"
	"fmt"
	"os"
)

// Error presents a useful message without exposing OS operations or temporary
// paths. Unwrap retains the original cause for errors.Is/As and callers.
type Error struct {
	Path    string
	Message string
	Cause   error
}

func (e *Error) Error() string { return fmt.Sprintf("%q：%s", e.Path, e.Message) }
func (e *Error) Unwrap() error { return e.Cause }

func ReadError(path string, err error) error {
	message := "无法读取文件，请检查文件是否可用及读取权限。"
	switch {
	case errors.Is(err, os.ErrNotExist):
		message = "文件不存在，请检查路径和文件名。"
	case errors.Is(err, os.ErrPermission):
		message = "没有读取权限，请调整文件或所在目录的权限。"
	}
	return &Error{Path: path, Message: message, Cause: err}
}
func WriteError(path string, err error) error {
	message := "无法写入文件，请检查目录权限、可用磁盘空间及目标是否为文件。"
	switch {
	case errors.Is(err, os.ErrExist):
		message = "文件已存在；如需覆盖，请加 --force。"
	case errors.Is(err, os.ErrNotExist):
		message = "输出目录不存在，请先创建目录或更换输出路径。"
	case errors.Is(err, os.ErrPermission):
		message = "没有写入权限，请更换输出目录或调整权限。"
	}
	return &Error{Path: path, Message: message, Cause: err}
}
