package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"resume-cli/internal/domain"
	"resume-cli/internal/fileio"
)

type Store struct {
	Dir string
	TTL time.Duration
	Now func() time.Time
}
type entry struct {
	Version int             `json:"version"`
	Created time.Time       `json:"created"`
	Data    json.RawMessage `json:"data"`
}

func (s Store) clock() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
func (s Store) path(key string) string { return filepath.Join(s.Dir, domain.Digest(key)+".json") }
func (s Store) Get(key string, out any) (bool, error) {
	if s.Dir == "" {
		return false, nil
	}
	b, e := fileio.Read(s.path(key), 2<<20)
	if errors.Is(e, os.ErrNotExist) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	var v entry
	if json.Unmarshal(b, &v) != nil || v.Version != 1 {
		return false, nil
	}
	ttl := s.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	age := s.clock().Sub(v.Created)
	if age < 0 || age > ttl {
		return false, nil
	}
	if json.Unmarshal(v.Data, out) != nil {
		return false, nil
	}
	return true, nil
}
func (s Store) Put(key string, v any) error {
	if s.Dir == "" {
		return nil
	}
	if e := os.MkdirAll(s.Dir, 0700); e != nil {
		return fileio.WriteError(s.Dir, e)
	}
	info, e := os.Lstat(s.Dir)
	if e != nil {
		return fileio.WriteError(s.Dir, e)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("缓存路径必须是目录，且不能是符号链接")
	}
	data, e := json.Marshal(v)
	if e != nil {
		return e
	}
	b, e := json.Marshal(entry{Version: 1, Created: s.clock(), Data: data})
	if e != nil {
		return e
	}
	return fileio.Write(s.path(key), b, true)
}
