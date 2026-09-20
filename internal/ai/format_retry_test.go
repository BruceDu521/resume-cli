package ai

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"resume-cli/internal/domain"
)

type sequenceGenerator struct {
	calls  int
	bodies []string
	err    error
}

func TestSourceGroundingRegeneration(t *testing.T) {
	d := domain.NewDocument("Alice\nGo")
	c := domain.Candidate{Resume: domain.Resume{Name: "Alice", Education: []domain.Education{}, Skills: []string{"Go"}}, Facts: []domain.Fact{{ID: "f1", Category: "skill", BlockID: "b2", Quote: "Rust"}}}
	bad, _ := json.Marshal(c)
	c.Facts[0].Quote = "Go"
	good, _ := json.Marshal(c)
	g := &sequenceGenerator{bodies: []string{string(bad), string(good)}}
	got, err := (Structurer{Generator: g}).Candidate(context.Background(), d)
	if err != nil || g.calls != 2 || got.Validate(d) != nil {
		t.Fatal(got, g.calls, err)
	}
}

func (s *sequenceGenerator) Identity() string { return "sequence" }
func (s *sequenceGenerator) Generate(_ context.Context, q Request) ([]byte, Usage, error) {
	s.calls++
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
