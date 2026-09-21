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
		{"valid initially", []string{good}, nil, 1, true},
		{"corrected", []string{bad, good}, nil, 2, true},
		{"second retry succeeds", []string{bad, bad, good}, nil, 3, true},
		{"third retry succeeds", []string{bad, bad, bad, good}, nil, 4, true},
		{"still malformed", []string{bad}, nil, 4, false},
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
			if tt.calls > 1 && stages[1] != "report_validation_retry" {
				t.Fatal(stages)
			}
			for _, request := range g.requests[1:] {
				if strings.Count(request.Instruction, "Validation issue:") != 1 {
					t.Fatal("correction instructions accumulated")
				}
			}
			if tt.name == "still malformed" && !strings.Contains(err.Error(), "after 3 corrective retries") {
				t.Fatal(err)
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

func TestValidationRetriesStopOnCancellation(t *testing.T) {
	for _, cancelAfterFirst := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		g := &sequenceGenerator{bodies: []string{`{}`}}
		if !cancelAfterFirst {
			cancel()
		}
		_, err := (Structurer{Generator: g, Observe: func(Usage) { cancel() }}).Evaluate(ctx, domain.NewDocument("Go"), "Go", "zh")
		cancel()
		wantCalls := 0
		if cancelAfterFirst {
			wantCalls = 1
		}
		if !errors.Is(err, context.Canceled) || g.calls != wantCalls {
			t.Fatal(g.calls, err)
		}
	}
}

func TestFieldValidationCanRecoverOnThirdRetry(t *testing.T) {
	good := validEvaluation()
	bad := good
	bad.Skill = 101
	g := &sequenceGenerator{bodies: []string{mustJSON(t, bad), mustJSON(t, bad), mustJSON(t, bad), mustJSON(t, good)}}
	got, err := (Structurer{Generator: g}).Evaluate(context.Background(), domain.NewDocument("Go"), "Go", "zh")
	if err != nil || g.calls != 4 || got.Skill != good.Skill {
		t.Fatal(g.calls, got, err)
	}
}

func TestExtractCanRecoverOnThirdRetry(t *testing.T) {
	good := domain.Resume{Name: "Synthetic", Education: []domain.Education{}, Skills: []string{"Go"}}
	g := &sequenceGenerator{bodies: []string{`{}`, `{}`, `{}`, mustJSON(t, good)}}
	got, err := (Structurer{Generator: g}).Extract(context.Background(), domain.NewDocument("Synthetic Go"))
	if err != nil || g.calls != 4 || got.Name != good.Name {
		t.Fatal(g.calls, got, err)
	}
}
