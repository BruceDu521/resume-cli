package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"resume-cli/internal/cache"
	"resume-cli/internal/domain"
)

type parser struct {
	d   domain.Document
	err error
}

func (p parser) Parse(context.Context, string) (domain.Document, error) { return p.d, p.err }

type structure struct {
	candidate atomic.Int32
	job       atomic.Int32
	err       error
}

func (s *structure) Candidate(ctx context.Context, d domain.Document) (domain.Candidate, error) {
	s.candidate.Add(1)
	if s.err != nil {
		return domain.Candidate{}, s.err
	}
	return domain.Candidate{Resume: domain.Resume{Name: "Alice", Education: []domain.Education{}, Skills: []string{"Go"}}, Facts: []domain.Fact{{ID: "f1", Category: "skill", BlockID: "b2", Quote: "Go"}}}, nil
}
func (s *structure) Job(context.Context, string) (domain.Job, error) {
	s.job.Add(1)
	return domain.Job{Requirements: []domain.Requirement{{ID: "r1", Category: "skill", Text: "Go", Required: true}}}, nil
}

type matcher struct{}

func (matcher) Match(context.Context, domain.Candidate, domain.Job) ([]domain.Judgment, error) {
	return []domain.Judgment{{RequirementID: "r1", Status: "satisfied", Score: 100, EvidenceID: "f1", Confidence: 1}}, nil
}
func TestReuseAcrossCommands(t *testing.T) {
	st := &structure{}
	s := Service{Parser: parser{d: domain.NewDocument("Alice\nGo")}, Structurer: st, Matcher: matcher{}, Cache: cache.Store{Dir: t.TempDir()}, Identity: "synthetic"}
	if _, e := s.Extract(context.Background(), "any"); e != nil {
		t.Fatal(e)
	}
	r, e := s.Score(context.Background(), "any", "Go", "zh")
	if e != nil || r.Overall != 100 {
		t.Fatal(r, e)
	}
	r, e = s.Score(context.Background(), "any", "Go", "en")
	if e != nil || r.Language != "en" {
		t.Fatal(e)
	}
	if st.candidate.Load() != 1 || st.job.Load() != 1 {
		t.Fatal("did not reuse", st.candidate.Load(), st.job.Load())
	}
}
func TestErrors(t *testing.T) {
	s := Service{Parser: parser{err: errors.New("parse failed")}}
	if _, e := s.Extract(context.Background(), "x"); e == nil {
		t.Fatal("parser failure")
	}
	if _, e := s.Score(context.Background(), "x", "Go", "xx"); e == nil {
		t.Fatal("lang")
	}
	s.Parser = parser{d: domain.NewDocument("Alice\nGo")}
	s.Structurer = &structure{err: errors.New("extraction failed")}
	s.Matcher = matcher{}
	if _, e := s.Score(context.Background(), "x", "Go", "zh"); e == nil {
		t.Fatal("extract error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.Score(ctx, "x", "Go", "zh"); e == nil {
		t.Fatal("cancel")
	}
}

type baseline struct{}

func (baseline) Evaluate(ctx context.Context, d domain.Document, jd, lang string) (domain.Candidate, domain.Job, []domain.Judgment, string, []string, error) {
	st := &structure{}
	c, _ := st.Candidate(ctx, d)
	j, _ := st.Job(ctx, jd)
	v, _ := (matcher{}).Match(ctx, c, j)
	return c, j, v, "Independent baseline", []string{"Question?"}, nil
}
func TestIndependentBaseline(t *testing.T) {
	s := Service{Parser: parser{d: domain.NewDocument("Alice\nGo")}, Baseline: baseline{}}
	r, e := s.Score(context.Background(), "any", "Go", "en")
	if e != nil || r.Comment != "Independent baseline" || r.Overall != 100 {
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

type emptyStructure struct{}

func (emptyStructure) Candidate(context.Context, domain.Document) (domain.Candidate, error) {
	return domain.Candidate{Resume: domain.Resume{Name: "Alice", Education: []domain.Education{}, Skills: []string{}}, Facts: []domain.Fact{}}, nil
}
func (emptyStructure) Job(context.Context, string) (domain.Job, error) {
	return domain.Job{Requirements: []domain.Requirement{{ID: "r", Category: "skill", Text: "Go", Required: true}}}, nil
}
func TestEmptyExtractionMustNotBecomeZeroScore(t *testing.T) {
	s := Service{Parser: parser{d: domain.NewDocument("Alice\nGo development")}, Structurer: emptyStructure{}}
	if _, err := s.Score(context.Background(), "any", "Go", "zh"); err == nil {
		t.Fatal("empty extraction scored")
	}
}
