package ai

import (
	"context"
	"errors"

	"resume-cli/internal/domain"
)

// Evaluate extracts and assesses original documents in one model request.
func (s Structurer) Evaluate(ctx context.Context, d domain.Document, jd, lang string) (domain.Candidate, domain.Job, []domain.Judgment, string, []string, error) {
	var v struct {
		Candidate domain.Candidate  `json:"candidate"`
		Job       domain.Job        `json:"job"`
		Judgments []domain.Judgment `json:"judgments"`
		Comment   string            `json:"comment"`
		Questions []string          `json:"interview_questions"`
	}
	js := object(map[string]any{"requirement_id": str(), "status": map[string]any{"type": "string", "enum": []string{"satisfied", "partial", "unmet", "unknown"}}, "score": map[string]any{"type": "number"}, "evidence_id": str(), "confidence": map[string]any{"type": "number"}})
	schema := object(map[string]any{"candidate": CandidateSchema(), "job": JobSchema(), "judgments": arr(js), "comment": str(), "interview_questions": arr(str())})
	validate := func() error {
		if err := v.Candidate.Validate(d); err != nil {
			return err
		}
		if err := v.Job.Validate(jd); err != nil {
			return err
		}
		if len(v.Candidate.Facts) == 0 || v.Comment == "" || len(v.Questions) < 1 || len(v.Questions) > 3 {
			return errors.New("incomplete single-model assessment")
		}
		for i := range v.Judgments {
			v.Judgments[i].Score = map[string]float64{"satisfied": 100, "partial": 50, "unmet": 0, "unknown": 0}[v.Judgments[i].Status]
		}
		_, err := domain.Aggregate(v.Candidate, v.Job, v.Judgments)
		return err
	}
	e := s.decodeChecked(ctx, Request{"assessment", safety + "\n" + requirementRules + "\nIndependently extract the candidate and JD requirements, then judge each requirement. Candidate facts must have unique IDs and category skill/experience/education. " + evidenceRules + " JD text must be verbatim and requirements have unique IDs. Each judgment references a requirement and a supporting/contradicting fact ID. Status satisfied=100, partial=50, unmet=0, unknown=0. Satisfied requires evidence of the requested level; personal projects can establish basic familiarity, but do not automatically establish professional experience. Partial means relevant evidence establishes some but not all required duration, scope or depth. Unmet requires explicit contradiction or denial; missing evidence is unknown, not inability. A shorter documented role is relevant partial evidence, not proof of total experience below a minimum. Never sum overlapping employment periods. Unknown has empty evidence_id; all other states require evidence. Include all judgments. Do not compute the overall score. For missing qualifications, describe missing resume evidence, not a confirmed lack of ability. Do not upgrade stated familiarity to expertise or assume unstated project outcomes. Write a concise comment and 1-3 interview questions in " + lang + ". Do not infer skill years from total employment duration.", map[string]any{"resume_blocks": d.Blocks, "jd": jd}, schema}, &v, validate)
	for i := range v.Judgments {
		v.Judgments[i].Score = map[string]float64{"satisfied": 100, "partial": 50, "unmet": 0, "unknown": 0}[v.Judgments[i].Status]
	}
	return v.Candidate, v.Job, v.Judgments, v.Comment, v.Questions, e
}
