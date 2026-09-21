package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"resume-cli/internal/domain"
	"resume-cli/internal/report"
)

func validEvaluation() report.Evaluation {
	return report.Evaluation{Overall: 75, Skill: 80, Experience: 65, Education: 100, Comment: "Backend experience is supported; frontend scope needs confirmation.", Questions: []string{"Describe your frontend responsibilities?"}}
}
func TestSingleScoreUsesCompleteTextWithoutCitationContract(t *testing.T) {
	d := domain.NewDocument("Project A: golang APIs.\nProject B: PostgreSQL backups.")
	lines := []string{}
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("Requirement %d: Go and PostgreSQL", i))
	}
	jd := strings.Join(lines, "\n")
	v := validEvaluation()
	g := &sequenceGenerator{bodies: []string{mustJSON(t, v)}}
	got, err := (Structurer{Generator: g}).Evaluate(context.Background(), d, jd, "zh")
	if err != nil || g.calls != 1 || got.Overall != 75 {
		t.Fatal(got, g.calls, err)
	}
	state := g.requests[0].State.(map[string]any)
	if len(state) != 2 || state["resume_text"] != d.Text || state["jd"] != jd {
		t.Fatal("input changed or truncated")
	}
	props := g.requests[0].Schema["properties"].(map[string]any)
	if len(props) != 6 || props["matches"] != nil || props["candidate"] != nil {
		t.Fatal(props)
	}
}
func TestScoreRejectsInvalidAndEmptyOutput(t *testing.T) {
	v := validEvaluation()
	for _, raw := range []string{`{"comment":"x","interview_questions":["Q?"]","matches":[]}`, `{"comment":"x","interview_questions":["Q?"],"matches":[]}`, `{}`, strings.Replace(mustJSON(t, v), `"overall_score":75`, `"overall_score":null`, 1), strings.Replace(mustJSON(t, v), `"overall_score":75`, `"overall_score":75.5`, 1)} {
		g := &sequenceGenerator{bodies: []string{raw}}
		if _, err := (Structurer{Generator: g}).Evaluate(context.Background(), domain.NewDocument("Go"), "Go", "zh"); err == nil || g.calls != 2 {
			t.Fatal("invalid output accepted", err)
		}
	}
	g := &sequenceGenerator{}
	if _, err := (Structurer{Generator: g}).Evaluate(context.Background(), domain.NewDocument("Go"), " ", "zh"); err == nil || g.calls != 0 {
		t.Fatal("empty JD sent")
	}
}

type correctionFailure struct {
	calls        int
	transportErr error
}

func (g *correctionFailure) Identity() string { return "offline" }
func (g *correctionFailure) Generate(context.Context, Request) ([]byte, Usage, error) {
	g.calls++
	if g.calls == 1 {
		return []byte(`{"private":"do-not-echo"}`), Usage{}, nil
	}
	return nil, Usage{}, g.transportErr
}
func TestCorrectionFailurePreservesOriginalReason(t *testing.T) {
	cause := errors.New("synthetic transport failure")
	g := &correctionFailure{transportErr: cause}
	_, err := (Structurer{Generator: g}).Evaluate(context.Background(), domain.NewDocument("Go"), "Go", "zh")
	if !errors.Is(err, cause) || !strings.Contains(err.Error(), "invalid JSON structure") || !strings.Contains(err.Error(), "corrective request failed") || strings.Contains(err.Error(), "do-not-echo") {
		t.Fatal(err)
	}
}
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
