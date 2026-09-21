package app

import (
	"context"
	"errors"
	"resume-cli/internal/cache"
	"resume-cli/internal/domain"
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

func (m *model) Candidate(ctx context.Context, d domain.Document) (domain.Candidate, error) {
	m.calls++
	if ctx.Err() != nil {
		return domain.Candidate{}, ctx.Err()
	}
	if m.err != nil {
		return domain.Candidate{}, m.err
	}
	c := domain.Candidate{Resume: domain.Resume{Name: "Alice", Education: []domain.Education{}, Skills: []string{"Go"}}, Facts: []domain.Fact{{ID: "f", Category: "skill", BlockID: "b2", Quote: "Go"}}}
	if m.empty {
		c.Facts = []domain.Fact{}
	}
	return c, nil
}
func (m *model) Evaluate(ctx context.Context, d domain.Document, jd, lang string) (domain.Candidate, domain.Job, []domain.Judgment, string, []string, error) {
	c, e := m.Candidate(ctx, d)
	j := domain.Job{Requirements: []domain.Requirement{{ID: "r", Category: "skill", Text: "Go", Required: true}}}
	v := []domain.Judgment{{RequirementID: "r", Status: "satisfied", Score: 100, EvidenceID: "f", Confidence: 1}}
	comment := "Evidence supports Go"
	if m.badReport {
		comment = ""
	}
	return c, j, v, comment, []string{"Describe your work?"}, e
}
func TestExtractCacheAndIndependentScore(t *testing.T) {
	m := &model{}
	s := Service{Parser: parser{d: domain.NewDocument("Alice\nGo")}, Extractor: m, Structurer: m, Evaluator: m, Cache: cache.Store{Dir: t.TempDir()}, Identity: "synthetic"}
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

func (m *model) Extract(ctx context.Context, d domain.Document) (domain.Resume, error) {
	c, e := m.Candidate(ctx, d)
	return c.Resume, e
}
func (m *model) Job(ctx context.Context, jd string) (domain.Job, error) {
	if ctx.Err() != nil {
		return domain.Job{}, ctx.Err()
	}
	return domain.Job{Requirements: []domain.Requirement{{ID: "r", Category: "skill", Text: "Go", Required: true}}}, m.err
}

type matcher struct{}

func (matcher) Match(context.Context, domain.Candidate, domain.Job) ([]domain.Judgment, error) {
	return []domain.Judgment{{RequirementID: "r", Status: "satisfied", Score: 100, EvidenceID: "f", Confidence: 1}}, nil
}
func TestOptionalJevPipeline(t *testing.T) {
	m := &model{}
	s := Service{Parser: parser{d: domain.NewDocument("Alice\nGo")}, Structurer: m, Matcher: matcher{}}
	r, e := s.Score(context.Background(), "any", "Go", "en")
	if e != nil || r.Overall != 100 {
		t.Fatal(r, e)
	}
}

type cancelStructure struct {
	started  chan struct{}
	finished chan struct{}
}

func (s cancelStructure) Candidate(ctx context.Context, _ domain.Document) (domain.Candidate, error) {
	<-s.started
	return domain.Candidate{}, errors.New("candidate failed")
}
func (s cancelStructure) Job(ctx context.Context, _ string) (domain.Job, error) {
	close(s.started)
	<-ctx.Done()
	close(s.finished)
	return domain.Job{}, ctx.Err()
}
func TestFailureCancelsAndJoinsWorkers(t *testing.T) {
	st := cancelStructure{make(chan struct{}), make(chan struct{})}
	s := Service{Parser: parser{d: domain.NewDocument("Alice")}, Structurer: st}
	_, err := s.Score(context.Background(), "any", "Go", "zh")
	if err == nil || err.Error() != "candidate failed" {
		t.Fatal(err)
	}
	select {
	case <-st.finished:
	default:
		t.Fatal("request worker outlived the command")
	}
}
