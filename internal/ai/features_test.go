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

func TestCandidateWrappedCitation(t *testing.T) {
	d := domain.NewDocument("Alice\nNo Rust\nproduction experience.")
	c := domain.Candidate{Resume: domain.Resume{Education: []domain.Education{}, Skills: []string{}}, Facts: []domain.Fact{{ID: "f1", Category: "experience", BlockID: "b2", EndBlockID: "b3", Quote: "No Rust production experience."}}}
	st := Structurer{Generator: fakeGenerator{value: c}}
	got, err := st.Candidate(context.Background(), d)
	if err != nil || got.Facts[0].EndBlockID != "b3" {
		t.Fatal(got, err)
	}
	c.Facts[0].EndBlockID = ""
	st.Generator = fakeGenerator{value: c}
	if _, err = st.Candidate(context.Background(), d); err == nil {
		t.Fatal("bad citation accepted")
	}
}
