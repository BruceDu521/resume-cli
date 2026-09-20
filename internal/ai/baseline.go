package ai

import (
	"context"

	"resume-cli/internal/domain"
)

// Evaluate uses original documents, not Jev answers, for an independent baseline.
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
	e := s.decode(ctx, Request{"baseline", safety + "\nIndependently extract the candidate and JD requirements, then judge each requirement. Candidate facts must quote a numbered document block, have unique IDs and category skill/experience/education. JD text must be verbatim and requirements have unique IDs. Each judgment references a requirement and a supporting/contradicting fact ID. Status satisfied=100, partial=50, unmet=0, unknown=0. Unknown has empty evidence_id; all other states require evidence. Include all judgments. Do not compute the overall score. Write a concise comment and 1-3 interview questions in " + lang + ". Do not infer skill years from total employment duration.", map[string]any{"resume_blocks": d.Blocks, "jd": jd}, schema}, &v)
	for i := range v.Judgments {
		v.Judgments[i].Score = map[string]float64{"satisfied": 100, "partial": 50, "unmet": 0, "unknown": 0}[v.Judgments[i].Status]
	}
	return v.Candidate, v.Job, v.Judgments, v.Comment, v.Questions, e
}
