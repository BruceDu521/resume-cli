package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Request struct {
	Stage       string
	Instruction string
	State       any
	Schema      map[string]any
}
type Generator interface {
	Generate(context.Context, Request) ([]byte, Usage, error)
	Identity() string
}
type Remote struct {
	Provider, Model, Key, BaseURL string
	HTTP                          *Transport
}

func (r *Remote) Identity() string { return r.Provider + ":" + r.Model + ":" + r.BaseURL }
func (r *Remote) Generate(ctx context.Context, q Request) ([]byte, Usage, error) {
	u := Usage{Stage: q.Stage, Provider: r.Provider, Model: r.Model}
	start := time.Now()
	if r.Key == "" {
		return nil, u, errors.New("missing API key for " + r.Provider)
	}
	if r.HTTP == nil {
		return nil, u, errors.New("missing HTTP transport")
	}
	state, e := json.Marshal(q.State)
	if e != nil {
		return nil, u, e
	}
	if len(state) > 200<<10 {
		return nil, u, errors.New("model input exceeds size limit")
	}
	if r.Provider == "gemini" {
		body := map[string]any{"model": r.Model, "system_instruction": q.Instruction, "input": string(state), "store": false, "response_format": map[string]any{"type": "text", "mime_type": "application/json", "schema": q.Schema}, "generation_config": map[string]any{"thinking_level": "low", "max_output_tokens": 12000}}
		var result struct {
			Status string `json:"status"`
			Model  string `json:"model"`
			Steps  []struct {
				Type    string `json:"type"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"steps"`
			Usage *struct {
				Input    *int `json:"total_input_tokens"`
				Output   *int `json:"total_output_tokens"`
				Thoughts int  `json:"total_thought_tokens"`
				Cached   int  `json:"total_cached_tokens"`
			} `json:"usage"`
		}
		u.Attempts, e = r.HTTP.Post(ctx, strings.TrimRight(r.BaseURL, "/")+"/interactions", map[string]string{"x-goog-api-key": r.Key}, body, &result)
		u.DurationMS = time.Since(start).Milliseconds()
		if e != nil {
			return nil, u, e
		}
		if result.Model != "" {
			u.Model = result.Model
		}
		if result.Usage != nil && result.Usage.Input != nil && result.Usage.Output != nil {
			u.Input = *result.Usage.Input
			u.Cached = result.Usage.Cached
			u.Output = *result.Usage.Output + result.Usage.Thoughts
			u.Known = true
		}
		if result.Status != "completed" {
			return nil, u, errors.New("Gemini response was not completed")
		}
		var text strings.Builder
		for _, step := range result.Steps {
			if step.Type == "model_output" {
				for _, p := range step.Content {
					if p.Type == "text" {
						text.WriteString(p.Text)
					}
				}
			}
		}
		estimate(&u, start)
		if text.Len() == 0 {
			return nil, u, errors.New("Gemini returned no text")
		}
		return []byte(text.String()), u, nil
	}
	schema, e := json.Marshal(q.Schema)
	if e != nil {
		return nil, u, e
	}
	body := map[string]any{"model": r.Model, "messages": []map[string]string{{"role": "system", "content": q.Instruction + "\nReturn JSON only matching this schema: " + string(schema)}, {"role": "user", "content": string(state)}}, "response_format": map[string]any{"type": "json_object"}}
	switch r.Provider {
	case "deepseek":
		body["thinking"] = map[string]string{"type": "disabled"}
		body["max_tokens"] = 12000
	case "openai":
		body["max_completion_tokens"] = 16000
		body["reasoning_effort"] = "low"
	case "kimi":
		if r.Model == "kimi-k3" || r.Model == "k3" || r.Model == "k3-256k" {
			body["max_completion_tokens"] = 16000
			body["reasoning_effort"] = "low"
			body["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "resume_analysis", "strict": true, "schema": q.Schema}}
		} else {
			body["max_tokens"] = 16000
		}
	default:
		return nil, u, errors.New("unsupported model provider")
	}
	var result struct {
		Model   string `json:"model"`
		Choices []struct {
			Finish  string `json:"finish_reason"`
			Message struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
		Usage *struct {
			Input   *int `json:"prompt_tokens"`
			Output  *int `json:"completion_tokens"`
			Hit     int  `json:"prompt_cache_hit_tokens"`
			Details struct {
				Cached int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
	}
	u.Attempts, e = r.HTTP.Post(ctx, strings.TrimRight(r.BaseURL, "/")+"/chat/completions", map[string]string{"Authorization": "Bearer " + r.Key}, body, &result)
	u.DurationMS = time.Since(start).Milliseconds()
	if e != nil {
		return nil, u, e
	}
	if result.Model != "" {
		u.Model = result.Model
	}
	if result.Usage != nil && result.Usage.Input != nil && result.Usage.Output != nil {
		u.Input = *result.Usage.Input
		u.Output = *result.Usage.Output
		u.Cached = max(result.Usage.Hit, result.Usage.Details.Cached)
		u.Known = true
	}
	estimate(&u, start)
	if endpoint, err := url.Parse(r.BaseURL); err == nil && r.Provider == "kimi" && strings.HasPrefix(endpoint.Path, "/coding") {
		u.CostUSD, u.PriceDate, u.CostComplete = nil, "", false
		u.CostNote = "Kimi Code subscription quota; no per-request USD billing estimate"
	}
	if len(result.Choices) != 1 || result.Choices[0].Finish != "stop" || result.Choices[0].Message.Refusal != "" || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return nil, u, errors.New("AI response refused, incomplete, or empty")
	}
	return []byte(result.Choices[0].Message.Content), u, nil
}

// ValidateEndpoint prevents accidental plaintext transmission of API keys.
func ValidateEndpoint(s string) error {
	u, e := url.Parse(s)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("AI endpoint must be an HTTPS URL without credentials, query or fragment")
	}
	return nil
}
