package report

import (
	"errors"
	"strings"

	"resume-cli/internal/domain"
)

// Evaluation is the public single-model scoring contract. Scores are model
// assessments, not independently verified measurements of candidate ability.
type Evaluation struct {
	Overall    int      `json:"overall_score"`
	Skill      int      `json:"skill_score"`
	Experience int      `json:"experience_score"`
	Education  int      `json:"education_score"`
	Comment    string   `json:"comment"`
	Questions  []string `json:"interview_questions"`
}

func (v Evaluation) Validate() error {
	for _, score := range []int{v.Overall, v.Skill, v.Experience, v.Education} {
		if score < 0 || score > 100 {
			return errors.New("assessment scores must be integers between 0 and 100")
		}
	}
	if strings.TrimSpace(v.Comment) == "" {
		return errors.New("assessment requires a nonempty comment")
	}
	if len(v.Questions) == 0 {
		return errors.New("assessment requires interview questions")
	}
	for _, q := range v.Questions {
		if strings.TrimSpace(q) == "" {
			return errors.New("interview questions must not be empty")
		}
	}
	return nil
}
func (v Evaluation) Result(lang string, mock bool) Result {
	return Result{Assessment: domain.Assessment{Overall: v.Overall, Skill: v.Skill, Experience: v.Experience, Education: v.Education, Policy: "model-assessment-v1"}, Comment: v.Comment, Questions: v.Questions, Language: lang, Mock: mock}
}
