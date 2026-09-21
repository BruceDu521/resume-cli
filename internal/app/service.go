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
type Extractor interface {
	Extract(context.Context, domain.Document) (domain.Resume, error)
}
type Structurer interface {
	Candidate(context.Context, domain.Document) (domain.Candidate, error)
	Job(context.Context, string) (domain.Job, error)
}
type Matcher interface {
	Match(context.Context, domain.Candidate, domain.Job) ([]domain.Judgment, error)
}
type Evaluator interface {
	Evaluate(context.Context, domain.Document, string, string) (domain.Candidate, domain.Job, []domain.Judgment, string, []string, error)
}
type Service struct {
	Parser     Parser
	Extractor  Extractor
	Structurer Structurer
	Matcher    Matcher
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
	key := "candidate:v7:" + s.Identity + ":" + d.Hash
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
func (s Service) job(ctx context.Context, text string) (domain.Job, error) {
	var j domain.Job
	key := "job:v5:" + s.Identity + ":" + domain.Digest(text)
	hit, e := s.Cache.Get(key, &j)
	if e != nil {
		return j, fmt.Errorf("job cache: %w", e)
	}
	if hit {
		if e = j.Validate(text); e == nil {
			if s.CacheHit != nil {
				s.CacheHit("job")
			}
			return j, nil
		}
	}
	j, e = s.Structurer.Job(ctx, text)
	if e == nil {
		e = j.Validate(text)
	}
	if e == nil {
		e = s.Cache.Put(key, j)
	}
	return j, e
}
func (s Service) Extract(ctx context.Context, path string) (domain.Resume, error) {
	d, e := s.Parse(ctx, path)
	if e != nil {
		return domain.Resume{}, e
	}
	var r domain.Resume
	key := "resume:v1:" + s.Identity + ":" + d.Hash
	hit, e := s.Cache.Get(key, &r)
	if e != nil {
		return r, fmt.Errorf("resume cache: %w", e)
	}
	if hit && r.Validate() == nil {
		if s.CacheHit != nil {
			s.CacheHit("resume")
		}
		return r, nil
	}
	r, e = s.Extractor.Extract(ctx, d)
	if e == nil {
		e = r.Validate()
	}
	if e == nil {
		e = s.Cache.Put(key, r)
	}
	return r, e
}
func (s Service) Score(ctx context.Context, path, jd, lang string) (report.Result, error) {
	if lang != "zh" && lang != "en" {
		return report.Result{}, errors.New("language must be zh or en")
	}
	d, e := s.Parse(ctx, path)
	if e != nil {
		return report.Result{}, e
	}
	var c domain.Candidate
	var job domain.Job
	var judgments []domain.Judgment
	var comment string
	var questions []string
	if s.Evaluator != nil {
		c, job, judgments, comment, questions, e = s.Evaluator.Evaluate(ctx, d, jd, lang)
		if e != nil {
			return report.Result{}, e
		}
		if c, e = c.Ground(d); e != nil {
			return report.Result{}, e
		}
		if e = job.Validate(jd); e != nil {
			return report.Result{}, e
		}
	} else {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		type result struct {
			c         domain.Candidate
			j         domain.Job
			e         error
			candidate bool
		}
		ch := make(chan result, 2)
		go func() { v, e := s.candidate(ctx, d); ch <- result{c: v, e: e, candidate: true} }()
		go func() { v, e := s.job(ctx, jd); ch <- result{j: v, e: e} }()
		// Drain both workers after cancellation so no request or usage callback
		// outlives the command (especially when collecting failure statistics).
		var firstErr error
		for range 2 {
			r := <-ch
			if r.e != nil && firstErr == nil {
				firstErr = r.e
				cancel()
			}
			if r.candidate {
				c = r.c
			} else {
				job = r.j
			}
		}
		if firstErr != nil {
			return report.Result{}, firstErr
		}
		if len(c.Facts) == 0 {
			return report.Result{}, errors.New("no assessable resume evidence; refusing to score an empty extraction")
		}

		judgments, e = s.Matcher.Match(ctx, c, job)
		if e != nil {
			return report.Result{}, e
		}
	}
	if len(c.Facts) == 0 {
		return report.Result{}, errors.New("no assessable resume evidence; refusing to score an empty extraction")
	}
	a, e := domain.Aggregate(c, job, judgments)
	if e != nil {
		return report.Result{}, e
	}
	r := report.Render(a, lang, s.Mock)
	if s.Evaluator != nil {
		if comment == "" || len(questions) < 1 || len(questions) > 3 {
			return report.Result{}, errors.New("invalid assessment report")
		}
		r.Comment = comment
		r.Questions = questions
	}
	return r, e
}
