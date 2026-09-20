package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"resume-cli/internal/domain"
)

func TestBaselineContract(t *testing.T) {
	d := domain.NewDocument("Alice\nGo")
	v := map[string]any{"candidate": domain.Candidate{Resume: domain.Resume{Name: "Alice", Education: []domain.Education{}, Skills: []string{"Go"}}, Facts: []domain.Fact{{ID: "f1", Category: "skill", BlockID: "b2", Quote: "Go"}}}, "job": domain.Job{Requirements: []domain.Requirement{{ID: "r1", Category: "skill", Text: "Go", Required: true}}}, "judgments": []domain.Judgment{{RequirementID: "r1", Status: "partial", Score: 90, EvidenceID: "f1", Confidence: .8}}, "comment": "Partial evidence", "interview_questions": []string{"Describe the work?"}}
	s := Structurer{Generator: fakeGenerator{value: v}}
	c, j, a, comment, q, e := s.Evaluate(context.Background(), d, "Go", "en")
	if e != nil || c.Validate(d) != nil || j.Validate("Go") != nil || a[0].Score != 50 || comment == "" || len(q) != 1 {
		t.Fatal(c, j, a, e)
	}
	// The common policy, not the baseline model, controls the final numbers.
	result, e := domain.Aggregate(c, j, a)
	if e != nil || result.Overall != 50 {
		t.Fatal(result, e)
	}
}
func TestNarrator(t *testing.T) {
	for _, good := range []bool{true, false} {
		v := map[string]any{"comment": "Grounded report", "interview_questions": []string{"What did you do?"}}
		if !good {
			v["interview_questions"] = []string{}
		}
		s := Structurer{Generator: fakeGenerator{value: v}}
		_, _, e := s.Narrate(context.Background(), domain.Assessment{}, "en")
		if (e == nil) != good {
			t.Fatal(e)
		}
	}
}
func TestMockRestrictions(t *testing.T) {
	m := Mock{}
	ctx := context.Background()
	if _, e := m.Candidate(ctx, domain.NewDocument("real resume")); e == nil {
		t.Fatal("mock pretended to understand arbitrary data")
	}
	if _, e := m.Job(ctx, "real JD"); e == nil {
		t.Fatal("mock arbitrary JD")
	}
	d := domain.NewDocument("RESUME_CLI_DEMO_V1\n林予安 | 杭州\nlin.yuan@example.com\n示例大学 | 软件工程 | 本科 | 2022\nGo / PostgreSQL\nKubernetes 部署")
	c, e := m.Candidate(ctx, d)
	if e != nil {
		t.Fatal(e)
	}
	j, e := m.Job(ctx, "RESUME_CLI_JD_V1\nGo / PostgreSQL\nKubernetes\n本科")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = m.Match(ctx, c, j); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, e = m.Candidate(ctx, d); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if _, e = m.Job(ctx, ""); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if _, e = m.Match(ctx, c, j); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
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
	s := Structurer{Generator: fakeGenerator{value: map[string]any{"resume": map[string]any{"education": []any{}, "skills": []any{}}, "facts": []any{}}}}
	if _, e := s.Candidate(context.Background(), domain.NewDocument("any")); e == nil {
		t.Fatal("missing keys accepted")
	}
	_, e := json.Marshal(CandidateSchema())
	if e != nil {
		t.Fatal(e)
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
