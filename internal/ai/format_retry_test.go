package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"resume-cli/internal/domain"
)

type sequenceGenerator struct {
	calls    int
	requests []Request
	bodies   []string
	err      error
}

func (s *sequenceGenerator) Identity() string { return "sequence" }
func (s *sequenceGenerator) Generate(_ context.Context, q Request) ([]byte, Usage, error) {
	s.calls++
	s.requests = append(s.requests, q)
	if s.err != nil {
		return nil, Usage{Stage: q.Stage}, s.err
	}
	return []byte(s.bodies[min(s.calls-1, len(s.bodies)-1)]), Usage{Stage: q.Stage}, nil
}
func TestBoundedFormatRegeneration(t *testing.T) {
	bad := `{"comment":"x","interview_questions":["Q?"],"additionalProperties":false}`
	good := `{"comment":"x","interview_questions":["Q?"]}`
	for _, tt := range []struct {
		name   string
		bodies []string
		err    error
		calls  int
		ok     bool
	}{
		{"corrected", []string{bad, good}, nil, 2, true},
		{"still malformed", []string{bad}, nil, 2, false},
		{"transport failure", nil, context.Canceled, 1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := &sequenceGenerator{bodies: tt.bodies, err: tt.err}
			stages := []string{}
			s := Structurer{Generator: g, Observe: func(u Usage) { stages = append(stages, u.Stage) }}
			var v struct {
				Comment   string   `json:"comment"`
				Questions []string `json:"interview_questions"`
			}
			err := s.decode(context.Background(), Request{Stage: "report"}, &v)
			if (err == nil) != tt.ok || g.calls != tt.calls || len(stages) != tt.calls {
				t.Fatal(g.calls, stages, err)
			}
			if tt.calls == 2 && stages[1] != "report_validation_retry" {
				t.Fatal(stages)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Fatal(err)
			}
		})
	}
}

func TestCorrectionIncludesSafeReasonOnly(t *testing.T) {
	v := validEvaluation()
	good := mustJSON(t, v)
	v.Skill = 101
	v.Comment = "private-invented-quote"
	g := &sequenceGenerator{bodies: []string{mustJSON(t, v), good}}
	_, err := (Structurer{Generator: g}).Evaluate(context.Background(), domain.NewDocument("Go"), "Go", "en")
	if err != nil || len(g.requests) != 2 {
		t.Fatal(err)
	}
	prompt := g.requests[1].Instruction
	if !strings.Contains(prompt, "integers between 0 and 100") || strings.Contains(prompt, "private-invented-quote") {
		t.Fatal("unsafe or missing validation reason")
	}
}
