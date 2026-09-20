package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Memory-only client: no socket is created and no default HTTP client is used.
type doFunc func(*http.Request) (*http.Response, error)

func (f doFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }
func response(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}
func TestTransportRetries(t *testing.T) {
	for _, status := range []int{429, 529, 502, 503, 504, 401, 422, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			tr := Transport{Attempts: 3, Client: doFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Header.Get("Authorization") != "Bearer synthetic" {
					t.Fatal("auth")
				}
				if calls == 1 {
					return response(status, "secret response must not leak"), nil
				}
				return response(200, `{"ok":true}`), nil
			})}
			var out map[string]bool
			n, e := tr.Post(context.Background(), "https://example.invalid", map[string]string{"Authorization": "Bearer synthetic"}, map[string]string{"x": "y"}, &out)
			retry := status == 429 || status == 529 || status == 502 || status == 503 || status == 504
			if retry {
				if e != nil || n != 2 || !out["ok"] {
					t.Fatal(n, e)
				}
			} else {
				if e == nil || n != 1 || strings.Contains(e.Error(), "secret") {
					t.Fatal(n, e)
				}
			}
		})
	}
}
func TestTransportBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
	}{{"badjson", "<html>", 200}, {"oversize", strings.Repeat("x", (2<<20)+1), 200}, {"retryLimit", "", 429}} {
		t.Run(tt.name, func(t *testing.T) {
			tr := Transport{Attempts: 100, Client: doFunc(func(*http.Request) (*http.Response, error) { return response(tt.status, tt.body), nil })}
			var v any
			n, e := tr.Post(context.Background(), "https://example.invalid", nil, nil, &v)
			if e == nil || n > 3 {
				t.Fatal(n, e)
			}
		})
	}
	tr := Transport{Client: doFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("key=secret") })}
	var out any
	_, e := tr.Post(context.Background(), "https://example.invalid", nil, nil, &out)
	if e == nil || strings.Contains(e.Error(), "secret") {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	tr = Transport{Attempts: 2, Delay: time.Second, Client: doFunc(func(*http.Request) (*http.Response, error) { cancel(); return response(429, ""), nil })}
	_, e = tr.Post(ctx, "https://example.invalid", nil, nil, &out)
	if !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	tr.Client = doFunc(func(*http.Request) (*http.Response, error) {
		r := response(429, "")
		r.Header.Set("Retry-After", "600")
		return r, nil
	})
	_, e = tr.Post(context.Background(), "https://example.invalid", nil, nil, &out)
	if e == nil {
		t.Fatal("unbounded wait")
	}
}
func TestEndpointValidation(t *testing.T) {
	for _, s := range []string{"http://example.com", "https://u:p@example.com", "https://example.com?k=secret", "https://example.com#fragment", "garbage"} {
		if ValidateEndpoint(s) == nil {
			t.Fatal(s)
		}
	}
	if e := ValidateEndpoint("https://api.example.com/v1"); e != nil {
		t.Fatal(e)
	}
}
