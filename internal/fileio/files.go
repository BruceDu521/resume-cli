package fileio

import (
	"errors"
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
		return nil, fmt.Errorf("cannot read input: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("input must be a regular file")
	}
	if info.Size() > limit {
		return nil, errors.New("input exceeds size limit")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("input exceeds size limit")
	}
	return b, nil
}
func Text(path string, limit int64) (string, error) {
	b, err := Read(path, limit)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", errors.New("text input must be UTF-8")
	}
	s := strings.TrimSpace(strings.TrimPrefix(string(b), "\ufeff"))
	if s == "" {
		return "", errors.New("text input is empty")
	}
	return s, nil
}

// Write commits a complete file. No-clobber uses link so an existing destination
// cannot be overwritten by a race between an existence check and a rename.
func Write(path string, data []byte, overwrite bool) error {
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
