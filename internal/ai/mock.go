package ai

import (
	"context"
	"errors"
	"strings"

	"resume-cli/internal/domain"
	"resume-cli/internal/report"
)

// Mock supports only the documented synthetic fixtures; it never pretends to
// infer arbitrary candidates. Tests for broader behavior use injected fakes.
type Mock struct{}

func (Mock) Candidate(ctx context.Context, d domain.Document) (domain.Candidate, error) {
	if e := ctx.Err(); e != nil {
		return domain.Candidate{}, e
	}
	if !strings.Contains(d.Text, "RESUME_CLI_DEMO_V1") {
		return domain.Candidate{}, errors.New("mock supports only testdata/resume-zh.pdf and resume-en.pdf")
	}
	r := domain.Resume{Name: "Lin Yuan", City: "Hangzhou", Email: "lin.yuan@example.com", Education: []domain.Education{{School: "Example University", Major: "Software Engineering", Degree: "Bachelor", GraduationTime: "2022"}}, Skills: []string{"Go", "PostgreSQL", "Kubernetes"}}
	if strings.Contains(d.Text, "林予安") {
		r.Name = "林予安"
		r.City = "杭州"
		r.Education = []domain.Education{{School: "示例大学", Major: "软件工程", Degree: "本科", GraduationTime: "2022"}}
	}
	c := domain.Candidate{Resume: r, Facts: []domain.Fact{}}
	for _, b := range d.Blocks {
		cat := ""
		id := ""
		switch {
		case strings.Contains(b.Text, "Go / PostgreSQL"):
			cat = "skill"
			id = "dev"
		case strings.Contains(b.Text, "Kubernetes"):
			cat = "experience"
			id = "ops"
		case strings.Contains(b.Text, "2022") && (strings.Contains(b.Text, "University") || strings.Contains(b.Text, "大学")):
			cat = "education"
			id = "edu"
		}
		if cat != "" {
			c.Facts = append(c.Facts, domain.Fact{ID: id, Category: cat, BlockID: b.ID, Quote: b.Text})
		}
	}
	return c, c.Validate(d)
}
func (Mock) Job(ctx context.Context, text string) (domain.Job, error) {
	if e := ctx.Err(); e != nil {
		return domain.Job{}, e
	}
	if !strings.Contains(text, "RESUME_CLI_JD_V1") {
		return domain.Job{}, errors.New("mock supports only testdata/jd.txt and jd-en.txt")
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 4 {
		return domain.Job{}, errors.New("invalid mock JD fixture")
	}
	j := domain.Job{Requirements: []domain.Requirement{{ID: "dev", Category: "skill", Text: lines[1], Required: true}, {ID: "ops", Category: "experience", Text: lines[2], Required: true}, {ID: "edu", Category: "education", Text: lines[3], Required: true}}}
	return j, j.Validate(text)
}
func (Mock) Match(ctx context.Context, c domain.Candidate, j domain.Job) ([]domain.Judgment, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	out := []domain.Judgment{}
	for _, r := range j.Requirements {
		s, score := "satisfied", 100.0
		if r.ID == "ops" {
			s, score = "partial", 50
		}
		out = append(out, domain.Judgment{RequirementID: r.ID, Status: s, Score: score, EvidenceID: r.ID, Confidence: 1})
	}
	return out, nil
}

func (m Mock) Evaluate(ctx context.Context, d domain.Document, jd, lang string) (domain.Candidate, domain.Job, []domain.Judgment, string, []string, error) {
	c, e := m.Candidate(ctx, d)
	if e != nil {
		return c, domain.Job{}, nil, "", nil, e
	}
	j, e := m.Job(ctx, jd)
	if e != nil {
		return c, j, nil, "", nil, e
	}
	v, e := m.Match(ctx, c, j)
	if e != nil {
		return c, j, v, "", nil, e
	}
	a, e := domain.Aggregate(c, j, v)
	if e != nil {
		return c, j, v, "", nil, e
	}
	r := report.Render(a, lang, true)
	return c, j, v, r.Comment, r.Questions, nil
}

func (m Mock) Extract(ctx context.Context, d domain.Document) (domain.Resume, error) {
	c, err := m.Candidate(ctx, d)
	return c.Resume, err
}
