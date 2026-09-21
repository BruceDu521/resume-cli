package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}
type Transport struct {
	Client   Doer
	Attempts int
	Delay    time.Duration
}

func NewTransport() *Transport {
	return &Transport{Client: &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirects are disabled") }}, Attempts: 3, Delay: 250 * time.Millisecond}
}

type HTTPError struct{ Status int }

func (e *HTTPError) Error() string { return fmt.Sprintf("AI service returned HTTP %d", e.Status) }
func (t *Transport) Post(ctx context.Context, url string, headers map[string]string, body any, out any) (int, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	attempts := t.Attempts
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 3 {
		attempts = 3
	}
	for n := 1; n <= attempts; n++ {
		req, e := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		if e != nil {
			return n, errors.New("invalid AI endpoint")
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		res, e := t.Client.Do(req)
		if e != nil {
			if ctx.Err() != nil {
				return n, ctx.Err()
			}
			return n, errors.New("AI request failed (network or timeout)")
		}
		raw, e := io.ReadAll(io.LimitReader(res.Body, (2<<20)+1))
		res.Body.Close()
		if e != nil {
			return n, errors.New("cannot read AI response")
		}
		if len(raw) > 2<<20 {
			return n, errors.New("AI response exceeds limit")
		}
		if res.StatusCode >= 200 && res.StatusCode < 300 {
			if err = json.Unmarshal(raw, out); err != nil {
				return n, errors.New("AI service returned invalid JSON envelope")
			}
			return n, nil
		}
		retry := res.StatusCode == 429 || res.StatusCode == 529 || res.StatusCode == 502 || res.StatusCode == 503 || res.StatusCode == 504
		if !retry || n == attempts {
			return n, &HTTPError{res.StatusCode}
		}
		delay := t.Delay * time.Duration(1<<(n-1))
		if v := res.Header.Get("Retry-After"); v != "" {
			if s, e := strconv.Atoi(v); e == nil && s >= 0 {
				delay = time.Duration(s) * time.Second
			} else if date, e := http.ParseTime(v); e == nil {
				delay = time.Until(date)
			}
		}
		if delay < 0 {
			delay = 0
		}
		if delay > 5*time.Second {
			return n, &HTTPError{res.StatusCode}
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return n, ctx.Err()
		case <-timer.C:
		}
	}
	return attempts, errors.New("AI request failed")
}

type Usage struct {
	Stage        string   `json:"stage"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	Input        int      `json:"input_tokens"`
	Cached       int      `json:"cached_input_tokens"`
	Output       int      `json:"output_tokens"`
	DurationMS   int64    `json:"duration_ms"`
	Attempts     int      `json:"attempts"`
	Repaired     bool     `json:"json_repaired"`
	Known        bool     `json:"usage_known"`
	CostUSD      *float64 `json:"estimated_cost_usd,omitempty"`
	PriceDate    string   `json:"price_date,omitempty"`
	CostComplete bool     `json:"cost_complete"`
	CostNote     string   `json:"cost_note,omitempty"`
}

func estimate(u *Usage, now time.Time) {
	if !u.Known || u.Input < 0 || u.Output < 0 || u.Cached < 0 || u.Cached > u.Input {
		return
	}
	var ip, cp, op float64
	switch u.Model {
	case "jev-1.13.0":
		ip, cp, op = .042, .042, 0
	case "gemini-3.8-flash":
		if !now.Before(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
			return
		}
		ip, cp, op = .75, .075, 3.75
	case "deepseek-flash":
		ip, cp, op = .30, .006, 1.20 // conservative peak estimate; not a billing quote
	case "gpt-6-astra":
		if u.Input > 272000 {
			return
		}
		ip, cp, op = 10, 1, 50
	case "kimi-k3":
		ip, cp, op = 3, .3, 15
	default:
		return
	}
	cost := (float64(u.Input-u.Cached)*ip + float64(u.Cached)*cp + float64(u.Output)*op) / 1e6
	if !math.IsNaN(cost) {
		u.CostUSD = &cost
		u.PriceDate = "2026-09-20"
		u.CostComplete = u.Attempts <= 1
		if u.Model == "kimi-k3" {
			// K3 separately bills cache writes. The current adapter records input,
			// hits and output but does not establish TTL-specific write usage.
			u.CostComplete = false
			u.CostNote = "input/output USD list-price estimate only; K3 cache-write charges are not included"
		}
	}
}
