package app

import (
	"context"
	"errors"
	"resume-cli/internal/cache"
	"resume-cli/internal/domain"
	"resume-cli/internal/report"
	"testing"
)

type parser struct {
	d   domain.Document
	err error
}

func (p parser) Parse(context.Context, string) (domain.Document, error) { return p.d, p.err }

type model struct {
	calls     int
	err       error
	empty     bool
	badReport bool
}

func (m *model) Evaluate(ctx context.Context, d domain.Document, jd, lang string) (report.Evaluation, error) {
	m.calls++
	if ctx.Err() != nil {
		return report.Evaluation{}, ctx.Err()
	}
	v := report.Evaluation{Overall: 100, Skill: 100, Experience: 100, Education: 100, Comment: "Evidence supports Go", Questions: []string{"Describe your work?"}}
	if m.empty {
		v.Questions = nil
	}
	if m.badReport {
		v.Comment = ""
	}
	return v, m.err
}

func TestExtractCacheAndIndependentScore(t *testing.T) {
	m := &model{}
	s := Service{Parser: parser{d: domain.NewDocument("Alice\nGo")}, Extractor: m, Evaluator: m, Cache: cache.Store{Dir: t.TempDir()}, Identity: "synthetic"}
	for range 2 {
		if _, err := s.Extract(context.Background(), "any"); err != nil {
			t.Fatal(err)
		}
	}
	if m.calls != 1 {
		t.Fatal("extract cache missed")
	}
	for _, lang := range []string{"zh", "en"} {
		r, err := s.Score(context.Background(), "any", "Go", lang)
		if err != nil || r.Overall != 100 || r.Language != lang || r.Comment != "Evidence supports Go" {
			t.Fatal(r, err)
		}
	}
	if m.calls != 3 {
		t.Fatal("score must run its complete analysis independently")
	}
}
func TestErrors(t *testing.T) {
	for _, m := range []*model{{err: errors.New("model failed")}, {empty: true}, {badReport: true}} {
		s := Service{Parser: parser{d: domain.NewDocument("Alice\nGo")}, Evaluator: m}
		if _, e := s.Score(context.Background(), "any", "Go", "zh"); e == nil {
			t.Fatal("invalid result accepted")
		}
	}
	s := Service{Parser: parser{err: errors.New("parse failed")}}
	if _, e := s.Extract(context.Background(), "any"); e == nil {
		t.Fatal("parser failure")
	}
	if _, e := s.Score(context.Background(), "any", "Go", "zh"); e == nil {
		t.Fatal("parser failure")
	}
	if _, e := s.Score(context.Background(), "any", "Go", "xx"); e == nil {
		t.Fatal("language failure")
	}
	m := &model{}
	s = Service{Parser: parser{d: domain.NewDocument("Alice\nGo")}, Evaluator: m}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.Score(ctx, "any", "Go", "zh"); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}

func (m *model) Extract(ctx context.Context, _ domain.Document) (domain.Resume, error) {
	m.calls++
	if ctx.Err() != nil {
		return domain.Resume{}, ctx.Err()
	}
	return domain.Resume{Name: "Alice", Education: []domain.Education{}, Skills: []string{"Go"}}, m.err
}
