package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"resume-cli/internal/domain"
)

func TestAnthropicRequestAndScoring(t *testing.T) {
	schema := assessmentSchema()
	transformed := anthropicSchema(schema)
	transformed["properties"].(map[string]any)["comment"].(map[string]any)["type"] = "integer"
	if schema["properties"].(map[string]any)["comment"].(map[string]any)["type"] != "string" {
		t.Fatal("nested schema mutated")
	}
	r := Remote{Provider: "anthropic", Model: "claude-sonnet-5", Key: "synthetic", BaseURL: "https://example.invalid/v1", HTTP: &Transport{Client: doFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/messages" || req.Header.Get("x-api-key") != "synthetic" || req.Header.Get("anthropic-version") != "2023-06-01" || req.Header.Get("Authorization") != "" {
			t.Fatal("wrong Anthropic request")
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		format := body["output_config"].(map[string]any)["format"].(map[string]any)
		props := format["schema"].(map[string]any)["properties"].(map[string]any)
		score := props["overall_score"].(map[string]any)
		if format["type"] != "json_schema" || score["minimum"] != nil || score["maximum"] != nil || body["model"] != "claude-sonnet-5" || body["system"] == nil || body["max_tokens"] != float64(16000) {
			t.Fatal(body)
		}
		return response(200, `{"type":"message","model":"claude-sonnet-5","stop_reason":"end_turn","content":[{"type":"thinking","thinking":"not returned"},{"type":"text","text":"{\"overall_score\":75,\"skill_score\":80,\"experience_score\":65,\"education_score\":100,\"comment\":\"Backend supported, frontend needs confirmation\",\"interview_questions\":[\"What frontend work?\"]}"}],"usage":{"input_tokens":2000,"output_tokens":400}}`), nil
	})}}
	var usage Usage
	got, err := (Structurer{Generator: &r, Observe: func(u Usage) { usage = u }}).Evaluate(context.Background(), domain.NewDocument("golang backend"), "Go and React", "en")
	if err != nil || got.Overall != 75 || !usage.Known || usage.Input != 2000 || usage.Output != 400 || usage.CostUSD == nil || *usage.CostUSD != .008 {
		t.Fatal(got, usage, err)
	}
	if schema["properties"].(map[string]any)["overall_score"].(map[string]any)["maximum"] != 100 {
		t.Fatal("shared schema mutated")
	}
}
func TestAnthropicFailuresAndUsage(t *testing.T) {
	for _, tt := range []struct {
		raw           string
		ok, known     bool
		input, cached int
	}{
		{`{"type":"message","stop_reason":"max_tokens","content":[{"type":"text","text":"{}"}],"usage":{"input_tokens":12,"output_tokens":20}}`, false, true, 12, 0},
		{`{"type":"message","stop_reason":"refusal","content":[{"type":"text","text":"no"}]}`, false, false, 0, 0},
		{`{"type":"message","stop_reason":"tool_use","content":[]}`, false, false, 0, 0},
		{`{"type":"message","stop_reason":"end_turn","content":[{"type":"thinking"}]}`, false, false, 0, 0},
		{`{"type":"message","stop_reason":"end_turn","content":[{"type":"text","text":"{}"}]}`, true, false, 0, 0},
		{`{"type":"message","stop_reason":"end_turn","content":[{"type":"text","text":"{}"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":20,"cache_creation_input_tokens":30}}`, true, true, 60, 20},
	} {
		r := Remote{Provider: "anthropic", Model: "claude-sonnet-5", Key: "synthetic", BaseURL: "https://example.invalid/v1", HTTP: &Transport{Client: doFunc(func(*http.Request) (*http.Response, error) { return response(200, tt.raw), nil })}}
		_, u, err := r.Generate(context.Background(), Request{Schema: ResumeSchema()})
		if (err == nil) != tt.ok || u.Known != tt.known || u.Input != tt.input || u.Cached != tt.cached {
			t.Fatal(u, err)
		}
		if tt.cached > 0 && (u.CostUSD != nil || u.CostNote == "") {
			t.Fatal("unpriced cache reported as complete", u)
		}
	}
}
func TestOpenAIStrictScoreContract(t *testing.T) {
	for _, model := range []string{"gpt-6-astra", "gpt-4.1"} {
		r := Remote{Provider: "openai", Model: model, Key: "synthetic", BaseURL: "https://example.invalid/v1", HTTP: &Transport{Client: doFunc(func(req *http.Request) (*http.Response, error) {
			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			format := body["response_format"].(map[string]any)
			if format["type"] != "json_schema" || format["json_schema"].(map[string]any)["strict"] != true || body["store"] != false {
				t.Fatal(body)
			}
			if model == "gpt-6-astra" && body["reasoning_effort"] != "low" {
				t.Fatal("missing reasoning control")
			}
			if model == "gpt-4.1" && body["reasoning_effort"] != nil {
				t.Fatal("unsupported reasoning parameter")
			}
			raw := mustJSON(t, validEvaluation())
			envelope := map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": raw}}}, "usage": map[string]any{"prompt_tokens": 2000, "completion_tokens": 400}}
			b, _ := json.Marshal(envelope)
			return response(200, string(b)), nil
		})}}
		got, err := (Structurer{Generator: &r}).Evaluate(context.Background(), domain.NewDocument("Go"), "Go", "en")
		if err != nil || got.Overall != 75 {
			t.Fatal(got, err)
		}
	}
}
