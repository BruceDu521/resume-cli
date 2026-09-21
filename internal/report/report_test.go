package report

import (
	"encoding/json"
	"testing"
)

func TestResultKeepsPublicJSONContract(t *testing.T) {
	v := Evaluation{Overall: 75, Skill: 80, Experience: 60, Education: 100, Comment: "Review", Questions: []string{"Explain?"}}
	b, err := json.Marshal(v.Result("en", false))
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"overall_score", "skill_score", "experience_score", "education_score", "comment", "interview_questions", "policy_version", "language", "mock"} {
		if _, ok := fields[key]; !ok {
			t.Fatal("missing public field", key)
		}
	}
	if len(fields) != 9 {
		t.Fatal("unexpected legacy fields", string(b))
	}
}
