package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func (r *Remote) generateAnthropic(ctx context.Context, q Request, state []byte, start time.Time) ([]byte, Usage, error) {
	u := Usage{Stage: q.Stage, Provider: "anthropic", Model: r.Model}
	schema, err := json.Marshal(q.Schema)
	if err != nil {
		return nil, u, err
	}
	body := map[string]any{
		"model": r.Model, "max_tokens": 16000,
		"system":        q.Instruction + "\nReturn JSON only matching this schema (including its value constraints): " + string(schema),
		"messages":      []map[string]string{{"role": "user", "content": string(state)}},
		"output_config": map[string]any{"format": map[string]any{"type": "json_schema", "schema": anthropicSchema(q.Schema)}},
	}
	var result struct {
		Type    string `json:"type"`
		Model   string `json:"model"`
		Stop    string `json:"stop_reason"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage *struct {
			Input    *int `json:"input_tokens"`
			Output   *int `json:"output_tokens"`
			Read     int  `json:"cache_read_input_tokens"`
			Creation int  `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}
	u.Attempts, err = r.HTTP.Post(ctx, strings.TrimRight(r.BaseURL, "/")+"/messages", map[string]string{"x-api-key": r.Key, "anthropic-version": "2023-06-01"}, body, &result)
	u.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		return nil, u, err
	}
	if result.Model != "" {
		u.Model = result.Model
	}
	if result.Usage != nil && result.Usage.Input != nil && result.Usage.Output != nil {
		u.Input = *result.Usage.Input + result.Usage.Read + result.Usage.Creation
		u.Cached = result.Usage.Read
		u.Output = *result.Usage.Output
		u.Known = true
		// No prompt-cache opt-in is sent. If a gateway adds caching, its write TTL
		// cannot be priced from these fields alone; report usage without a false bill.
		if result.Usage.Creation == 0 && result.Usage.Read == 0 {
			estimate(&u, start)
			if u.CostUSD != nil {
				u.PriceDate = "2026-09-21"
			}
		} else {
			u.CostNote = "Anthropic cache usage present; cache pricing not estimated"
		}
	}
	if result.Type != "message" || result.Stop != "end_turn" {
		return nil, u, errors.New("Anthropic response refused, incomplete, or unexpected")
	}
	var text strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return nil, u, errors.New("Anthropic returned no text")
	}
	return []byte(text.String()), u, nil
}

// Claude's structured-output grammar does not support numeric bounds. Keep
// those in the instruction and local validation, without mutating shared schemas.
func anthropicSchema(schema map[string]any) map[string]any {
	out := make(map[string]any, len(schema))
	for k, v := range schema {
		switch k {
		case "minimum", "maximum":
			continue
		case "properties":
			props := map[string]any{}
			for name, property := range v.(map[string]any) {
				props[name] = anthropicSchema(property.(map[string]any))
			}
			out[k] = props
		case "items":
			out[k] = anthropicSchema(v.(map[string]any))
		default:
			out[k] = v
		}
	}
	return out
}
