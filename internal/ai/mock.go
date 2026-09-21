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

func (Mock) Extract(ctx context.Context, d domain.Document) (domain.Resume, error) {
	if e := ctx.Err(); e != nil {
		return domain.Resume{}, e
	}
	if !strings.Contains(d.Text, "RESUME_CLI_DEMO_V1") {
		return domain.Resume{}, errors.New("mock supports only synthetic resumes; export them with: resume-cli samples demo-inputs")
	}
	r := domain.Resume{Name: "Lin Yuan", City: "Hangzhou", Email: "lin.yuan@example.com", Education: []domain.Education{{School: "Example University", Major: "Software Engineering", Degree: "Bachelor", GraduationTime: "2022"}}, Skills: []string{"Go", "PostgreSQL", "Kubernetes"}}
	if strings.Contains(d.Text, "林予安") {
		r.Name = "林予安"
		r.City = "杭州"
		r.Education = []domain.Education{{School: "示例大学", Major: "软件工程", Degree: "本科", GraduationTime: "2022"}}
	}
	return r, nil
}
func (m Mock) Evaluate(ctx context.Context, d domain.Document, jd, lang string) (report.Evaluation, error) {
	if _, err := m.Extract(ctx, d); err != nil {
		return report.Evaluation{}, err
	}
	if !strings.Contains(jd, "RESUME_CLI_JD_V1") || len(strings.Split(strings.TrimSpace(jd), "\n")) != 4 {
		return report.Evaluation{}, errors.New("mock supports only synthetic JDs; export them with: resume-cli samples demo-inputs")
	}
	v := report.Evaluation{Overall: 83, Skill: 100, Experience: 50, Education: 100}
	switch lang {
	case "zh":
		v.Comment = "合成演示：Go/PostgreSQL 开发及本科学历符合要求，Kubernetes 独立生产运维经验需要进一步确认。"
		v.Questions = []string{"请介绍你在 Kubernetes 部署和生产运维中实际承担的职责。"}
	case "en":
		v.Comment = "Synthetic demo: Go/PostgreSQL development and education meet the requirements; independent Kubernetes production operations need confirmation."
		v.Questions = []string{"Describe your responsibilities in Kubernetes deployment and production operations."}
	default:
		return report.Evaluation{}, errors.New("language must be zh or en")
	}
	return v, nil
}
