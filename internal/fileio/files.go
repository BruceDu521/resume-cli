package fileio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func Read(path string, limit int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, ReadError(path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, &Error{Path: path, Message: "路径不是普通文件，请提供文件路径，不要使用目录或设备。"}
	}
	if info.Size() > limit {
		return nil, &Error{Path: path, Message: fmt.Sprintf("文件过大（上限 %d 字节），请缩小文件后重试。", limit)}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, ReadError(path, err)
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, ReadError(path, err)
	}
	if int64(len(b)) > limit {
		return nil, &Error{Path: path, Message: fmt.Sprintf("文件过大（上限 %d 字节），请缩小文件后重试。", limit)}
	}
	return b, nil
}
func Text(path string, limit int64) (string, error) {
	b, err := Read(path, limit)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(string(b), "%PDF-") {
		return "", &Error{Path: path, Message: "需要纯文本岗位描述，不能直接读取 PDF；请先转为 UTF-8 的 .txt 文件。"}
	}
	if !utf8.Valid(b) {
		return "", &Error{Path: path, Message: "不是有效的 UTF-8 文本，请将岗位描述另存为 UTF-8 的 .txt 文件。"}
	}
	s := strings.TrimSpace(strings.TrimPrefix(string(b), "\ufeff"))
	if s == "" {
		return "", &Error{Path: path, Message: "文件为空或仅含空白，请填写岗位描述后重试。"}
	}
	return s, nil
}

// Write commits a complete file. No-clobber uses link so an existing destination
// cannot be overwritten by a race between an existence check and a rename.
func Write(path string, data []byte, overwrite bool) (resultErr error) {
	defer func() {
		if resultErr != nil {
			resultErr = WriteError(path, resultErr)
		}
	}()
	f, err := os.CreateTemp(filepath.Dir(path), ".resume-cli-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if overwrite {
		return os.Rename(tmp, path)
	}
	return os.Link(tmp, path)
}
