package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"resume-cli/internal/domain"
)

func TestProviderContracts(t *testing.T) {
	for _, provider := range []string{"gemini", "deepseek", "openai", "kimi"} {
		t.Run(provider, func(t *testing.T) {
			r := Remote{Provider: provider, Model: "model", Key: "synthetic", BaseURL: "https://example.invalid/v1", HTTP: &Transport{Client: doFunc(func(req *http.Request) (*http.Response, error) {
				var body map[string]any
				if e := json.NewDecoder(req.Body).Decode(&body); e != nil {
					t.Fatal(e)
				}
				if body["model"] != "model" {
					t.Fatal(body)
				}
				if provider == "gemini" {
					if req.URL.Path != "/v1/interactions" || req.Header.Get("x-goog-api-key") != "synthetic" || body["store"] != false || body["system_instruction"] == nil {
						t.Fatal(body)
					}
					return response(200, `{"status":"completed","model":"gemini-3.8-flash","steps":[{"type":"model_output","content":[{"type":"text","text":"{}"}]}],"usage":{"total_input_tokens":100,"total_output_tokens":20,"total_thought_tokens":10,"total_cached_tokens":5}}`), nil
				}
				if req.URL.Path != "/v1/chat/completions" || req.Header.Get("Authorization") != "Bearer synthetic" {
					t.Fatal(req.URL)
				}
				if provider == "deepseek" && body["thinking"].(map[string]any)["type"] != "disabled" {
					t.Fatal(body)
				}
				if provider == "openai" && body["reasoning_effort"] != "low" {
					t.Fatal(body)
				}
				return response(200, `{"model":"model","choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":{"cached_tokens":5}}}`), nil
			})}}
			b, u, e := r.Generate(context.Background(), Request{Stage: "candidate", Instruction: "rules", State: map[string]string{"x": "y"}, Schema: object(map[string]any{})})
			if e != nil || string(b) != "{}" || !u.Known || u.Input != 100 || u.Cached != 5 {
				t.Fatal(string(b), u, e)
			}
			if provider == "gemini" && u.Output != 30 {
				t.Fatal("thinking tokens not billed", u)
			}
		})
	}
}
func TestProviderFailures(t *testing.T) {
	for _, raw := range []string{`{"choices":[]}`, `{"choices":[{"finish_reason":"length","message":{"content":"{}"}}]}`, `{"choices":[{"finish_reason":"stop","message":{"content":"{}","refusal":"no"}}]}`} {
		r := Remote{Provider: "openai", Model: "x", Key: "synthetic", BaseURL: "https://example.invalid", HTTP: &Transport{Client: doFunc(func(*http.Request) (*http.Response, error) { return response(200, raw), nil })}}
		if _, _, e := r.Generate(context.Background(), Request{}); e == nil {
			t.Fatal(raw)
		}
	}
	r := Remote{Provider: "openai"}
	if _, _, e := r.Generate(context.Background(), Request{}); e == nil {
		t.Fatal("missing key")
	}
}

func TestK3StrictSchemaAndReasoningContract(t *testing.T) {
	r := Remote{Provider: "kimi", Model: "kimi-k3", Key: "synthetic", BaseURL: "https://example.invalid", HTTP: &Transport{Client: doFunc(func(req *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		format := body["response_format"].(map[string]any)
		if body["reasoning_effort"] != "low" || body["max_completion_tokens"] != float64(16000) || body["max_tokens"] != nil || body["thinking"] != nil || format["type"] != "json_schema" || format["json_schema"].(map[string]any)["strict"] != true {
			t.Fatal(body)
		}
		return response(200, `{"model":"kimi-k3","choices":[{"finish_reason":"stop","message":{"content":"{}","reasoning_content":"not output"}}],"usage":{"prompt_tokens":100,"completion_tokens":20}}`), nil
	})}}
	b, _, err := r.Generate(context.Background(), Request{Schema: object(map[string]any{})})
	if err != nil || string(b) != "{}" {
		t.Fatal(string(b), err)
	}
}
func TestEstimate(t *testing.T) {
	u := Usage{Model: "gemini-3.8-flash", Input: 1000000, Output: 1000000, Known: true}
	estimate(&u, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	if u.CostUSD == nil || *u.CostUSD != 4.5 {
		t.Fatal(u)
	}
	u.CostUSD = nil
	estimate(&u, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	if u.CostUSD != nil {
		t.Fatal("stale promotional price")
	}
	u = Usage{Model: "kimi-k3", Input: 1000, Output: 100, Known: true, Attempts: 1}
	estimate(&u, time.Now())
	if u.CostUSD == nil || u.CostComplete || u.CostNote == "" {
		t.Fatal("K3 cache-write costs must not be reported as complete", u)
	}
}

type fakeGenerator struct {
	value any
	err   error
}

func (f fakeGenerator) Generate(_ context.Context, _ Request) ([]byte, Usage, error) {
	b, _ := json.Marshal(f.value)
	return b, Usage{}, f.err
}
func (fakeGenerator) Identity() string { return "fake" }
func TestStructureValidation(t *testing.T) {
	d := domain.NewDocument("Lin Yuan\nGo development")
	c := domain.Candidate{Resume: domain.Resume{Name: "Lin Yuan", Education: []domain.Education{}, Skills: []string{"Go"}}, Facts: []domain.Fact{{ID: "f1", Category: "skill", BlockID: "b2", Quote: "Go development"}}}
	observed := false
	s := Structurer{Generator: fakeGenerator{value: c}, Observe: func(Usage) { observed = true }}
	if _, e := s.Candidate(context.Background(), d); e != nil || !observed {
		t.Fatal(e)
	}
	c.Facts[0].Quote = "made up"
	s.Generator = fakeGenerator{value: c}
	if _, e := s.Candidate(context.Background(), d); e == nil {
		t.Fatal("hallucinated evidence")
	}

}

func TestKimiCodeDoesNotPretendSubscriptionIsPayPerToken(t *testing.T) {
	r := Remote{Provider: "kimi", Model: "k3", Key: "synthetic", BaseURL: "https://api.kimi.com/coding/v1", HTTP: &Transport{Client: doFunc(func(req *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "k3" || body["reasoning_effort"] != "low" || body["max_completion_tokens"] == nil {
			t.Fatal(body)
		}
		return response(200, `{"model":"kimi-k3","choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{"prompt_tokens":100,"completion_tokens":20}}`), nil
	})}}
	_, u, err := r.Generate(context.Background(), Request{Schema: object(map[string]any{})})
	if err != nil || !u.Known || u.CostUSD != nil || u.CostComplete || u.CostNote == "" {
		t.Fatal(u, err)
	}
}
