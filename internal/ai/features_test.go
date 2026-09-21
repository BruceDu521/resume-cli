package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"resume-cli/internal/domain"
	"resume-cli/internal/report"
)

func TestAssessmentContract(t *testing.T) {
	d := domain.NewDocument("Alice\nGo")
	v := report.Evaluation{Overall: 50, Skill: 50, Experience: 50, Education: 50, Comment: "Partial evidence", Questions: []string{"Describe the work?"}}
	good, _ := json.Marshal(v)
	bad := strings.Replace(string(good), `"skill_score":50`, `"skill_score":101`, 1)
	g := &sequenceGenerator{bodies: []string{bad, string(good)}}
	result, e := (Structurer{Generator: g}).Evaluate(context.Background(), d, "Go", "en")
	if e != nil || result.Overall != 50 || result.Skill != 50 || g.calls != 2 {
		t.Fatal(result, g.calls, e)
	}
	if g.requests[1].Stage != "assessment_validation_retry" {
		t.Fatal("missing bounded correction")
	}
}
func TestMockRestrictions(t *testing.T) {
	m := Mock{}
	ctx := context.Background()
	d := domain.NewDocument("RESUME_CLI_DEMO_V1\n林予安")
	jd := "RESUME_CLI_JD_V1\nGo / PostgreSQL\nKubernetes\n本科"
	if _, err := m.Extract(ctx, domain.NewDocument("real resume")); err == nil {
		t.Fatal("arbitrary resume accepted")
	}
	if _, err := m.Evaluate(ctx, d, "real JD", "zh"); err == nil {
		t.Fatal("arbitrary JD accepted")
	}
	if _, err := m.Evaluate(ctx, d, jd, "zh"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := m.Extract(ctx, d); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := m.Evaluate(ctx, d, jd, "en"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestRemoteEmptyUsageAndIncomplete(t *testing.T) {
	for _, raw := range []string{`{"status":"incomplete"}`, `{"status":"completed","steps":[]}`} {
		r := Remote{Provider: "gemini", Model: "x", Key: "synthetic", BaseURL: "https://example.invalid", HTTP: &Transport{Client: doFunc(func(*http.Request) (*http.Response, error) { return response(200, raw), nil })}}
		if _, u, e := r.Generate(context.Background(), Request{}); e == nil || u.Known {
			t.Fatal(u, e)
		}
	}
	r := Remote{Provider: "gemini", Model: "x", Key: "synthetic", BaseURL: "https://example.invalid", HTTP: NewTransport()}
	if !strings.Contains(r.Identity(), "gemini") {
		t.Fatal(r.Identity())
	}
	if _, _, e := r.Generate(context.Background(), Request{}); e == nil {
		t.Fatal("network guard failed")
	}
	if e := r.HTTP.Client.(*http.Client).CheckRedirect(nil, nil); e == nil {
		t.Fatal("redirect allowed")
	}
}
func TestMissingRequiredNestedField(t *testing.T) {
	s := Structurer{Generator: fakeGenerator{value: map[string]any{"name": "Alice", "phone": "", "email": "", "city": "", "skills": []any{}, "education": []any{map[string]any{"school": "Example", "degree": "Bachelor", "graduation_time": "2022"}}}}}
	if _, err := s.Extract(context.Background(), domain.NewDocument("Alice")); err == nil {
		t.Fatal("missing nested major accepted")
	}
}

func TestMissingTokenCountsAreUnknown(t *testing.T) {
	for _, provider := range []string{"gemini", "openai"} {
		raw := `{"choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{}}`
		if provider == "gemini" {
			raw = `{"status":"completed","steps":[{"type":"model_output","content":[{"type":"text","text":"{}"}]}],"usage":{}}`
		}
		r := Remote{Provider: provider, Model: "model", Key: "synthetic", BaseURL: "https://example.invalid", HTTP: &Transport{Client: doFunc(func(*http.Request) (*http.Response, error) { return response(200, raw), nil })}}
		_, u, err := r.Generate(context.Background(), Request{})
		if err != nil || u.Known || u.CostUSD != nil || u.CostComplete {
			t.Fatal(u, err)
		}
	}
}
