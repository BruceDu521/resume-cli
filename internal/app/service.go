package app

import (
	"context"
	"errors"
	"fmt"

	"resume-cli/internal/cache"
	"resume-cli/internal/domain"
	"resume-cli/internal/report"
)

type Parser interface {
	Parse(context.Context, string) (domain.Document, error)
}
type Structurer interface {
	Candidate(context.Context, domain.Document) (domain.Candidate, error)
}
type Evaluator interface {
	Evaluate(context.Context, domain.Document, string, string) (domain.Candidate, domain.Job, []domain.Judgment, string, []string, error)
}
type Service struct {
	Parser     Parser
	Structurer Structurer
	Evaluator  Evaluator
	Cache      cache.Store
	Identity   string
	Mock       bool
	CacheHit   func(string)
}

func (s Service) Parse(ctx context.Context, path string) (domain.Document, error) {
	return s.Parser.Parse(ctx, path)
}
func (s Service) candidate(ctx context.Context, d domain.Document) (domain.Candidate, error) {
	var c domain.Candidate
	key := "candidate:v6:" + s.Identity + ":" + d.Hash
	hit, e := s.Cache.Get(key, &c)
	if e != nil {
		return c, fmt.Errorf("candidate cache: %w", e)
	}
	if hit {
		if c, e = c.Ground(d); e == nil {
			if s.CacheHit != nil {
				s.CacheHit("candidate")
			}
			return c, nil
		}
	}
	c, e = s.Structurer.Candidate(ctx, d)
	if e == nil {
		c, e = c.Ground(d)
	}
	if e == nil {
		e = s.Cache.Put(key, c)
	}
	return c, e
}
func (s Service) Extract(ctx context.Context, path string) (domain.Resume, error) {
	d, e := s.Parse(ctx, path)
	if e != nil {
		return domain.Resume{}, e
	}
	c, e := s.candidate(ctx, d)
	return c.Resume, e
}
func (s Service) Score(ctx context.Context, path, jd, lang string) (report.Result, error) {
	if lang != "zh" && lang != "en" {
		return report.Result{}, errors.New("language must be zh or en")
	}
	d, e := s.Parse(ctx, path)
	if e != nil {
		return report.Result{}, e
	}
	c, job, judgments, comment, questions, e := s.Evaluator.Evaluate(ctx, d, jd, lang)
	if e != nil {
		return report.Result{}, e
	}
	if c, e = c.Ground(d); e != nil {
		return report.Result{}, e
	}
	if e = job.Validate(jd); e != nil {
		return report.Result{}, e
	}
	if len(c.Facts) == 0 {
		return report.Result{}, errors.New("no assessable resume evidence; refusing to score an empty extraction")
	}
	a, e := domain.Aggregate(c, job, judgments)
	if e != nil {
		return report.Result{}, e
	}
	if comment == "" || len(questions) < 1 || len(questions) > 3 {
		return report.Result{}, errors.New("invalid assessment report")
	}
	r := report.Render(a, lang, s.Mock)
	r.Comment, r.Questions = comment, questions
	return r, nil
}
