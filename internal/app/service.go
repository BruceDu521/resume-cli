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
type Evaluator interface {
	Evaluate(context.Context, domain.Document, string, string) (report.Evaluation, error)
}
type Service struct {
	Parser    Parser
	Extractor Extractor
	Evaluator Evaluator
	Cache     cache.Store
	Identity  string
	Mock      bool
	CacheHit  func(string)
}

func (s Service) Parse(ctx context.Context, path string) (domain.Document, error) {
	return s.Parser.Parse(ctx, path)
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
	v, err := s.Evaluator.Evaluate(ctx, d, jd, lang)
	if err != nil {
		return report.Result{}, err
	}
	if err = v.Validate(); err != nil {
		return report.Result{}, err
	}
	return v.Result(lang, s.Mock), nil
}
