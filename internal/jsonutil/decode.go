// Package jsonutil provides deliberately bounded syntax repair. It never invents data.
package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

const MaxBytes = 2 << 20

func Decode(data []byte, out any) (bool, error) {
	if len(data) > MaxBytes {
		return false, errors.New("JSON response exceeds limit")
	}
	repaired := false
	s := strings.TrimSpace(strings.TrimPrefix(string(data), "\ufeff"))
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) < 3 || (strings.TrimSpace(lines[0]) != "```json" && strings.TrimSpace(lines[0]) != "```") || strings.TrimSpace(lines[len(lines)-1]) != "```" {
			return false, errors.New("invalid JSON code fence")
		}
		s = strings.Join(lines[1:len(lines)-1], "\n")
		repaired = true
	}
	if !json.Valid([]byte(s)) {
		fixed := trailingCommas(s)
		if fixed != s {
			s = fixed
			repaired = true
		}
	}
	// Token scan rejects duplicate object keys and null values before struct decoding.
	scan := json.NewDecoder(strings.NewReader(s))
	scan.UseNumber()
	if err := value(scan, 0); err != nil {
		return repaired, err
	}
	if _, err := scan.Token(); err != io.EOF {
		return repaired, errors.New("expected exactly one JSON value")
	}
	d := json.NewDecoder(bytes.NewBufferString(s))
	var shape any
	if err := json.Unmarshal([]byte(s), &shape); err != nil {
		return repaired, err
	}
	if err := required(shape, reflect.TypeOf(out)); err != nil {
		return repaired, err
	}
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return repaired, err
	}
	return repaired, nil
}

func required(v any, typ reflect.Type) error {
	if typ == nil {
		return errors.New("JSON target must not be nil")
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Struct:
		obj, ok := v.(map[string]any)
		if !ok {
			return errors.New("expected JSON object")
		}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "-" || !f.IsExported() {
				continue
			}
			name := tag[0]
			if name == "" {
				name = f.Name
			}
			item, exists := obj[name]
			optional := strings.Contains(f.Tag.Get("json"), "omitempty")
			if !exists {
				if optional {
					continue
				}
				return fmt.Errorf("missing required field %s", name)
			}
			if err := required(item, f.Type); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		a, ok := v.([]any)
		if !ok {
			return errors.New("expected JSON array")
		}
		for _, item := range a {
			if err := required(item, typ.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}
func value(d *json.Decoder, depth int) error {
	if depth > 64 {
		return errors.New("JSON nesting exceeds limit")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	if t == nil {
		return errors.New("null is not allowed; use empty string or array")
	}
	if delim, ok := t.(json.Delim); ok {
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				key, ok := k.(string)
				if !ok || seen[key] {
					return errors.New("duplicate or invalid JSON key")
				}
				seen[key] = true
				if e = value(d, depth+1); e != nil {
					return e
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim('}') {
				return errors.New("unterminated JSON object")
			}
		case '[':
			for d.More() {
				if e := value(d, depth+1); e != nil {
					return e
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim(']') {
				return errors.New("unterminated JSON array")
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
	}
	return nil
}
func trailingCommas(s string) string {
	var b strings.Builder
	quoted, escaped := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quoted {
			b.WriteByte(c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				quoted = false
			}
			continue
		}
		if c == '"' {
			quoted = true
		}
		if c == ',' {
			j := i + 1
			for j < len(s) && strings.ContainsRune(" \r\n\t", rune(s[j])) {
				j++
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}
