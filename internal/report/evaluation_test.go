package report

import "testing"

func TestEvaluationValidation(t *testing.T) {
	base := Evaluation{Overall: 75, Skill: 80, Experience: 60, Education: 100, Comment: "Supported with gaps", Questions: []string{"Explain the gap?"}}
	for _, update := range []func(*Evaluation){func(v *Evaluation) { v.Overall = -1 }, func(v *Evaluation) { v.Skill = 101 }, func(v *Evaluation) { v.Experience = -1 }, func(v *Evaluation) { v.Education = 101 }, func(v *Evaluation) { v.Comment = " " }, func(v *Evaluation) { v.Questions = nil }, func(v *Evaluation) { v.Questions = []string{" "} }} {
		v := base
		update(&v)
		if v.Validate() == nil {
			t.Fatal("invalid report accepted", v)
		}
	}
	for _, n := range []int{0, 100} {
		v := base
		v.Overall = n
		v.Skill = n
		v.Experience = n
		v.Education = n
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	r := base.Result("zh", false)
	if r.Policy != "model-assessment-v1" || r.Overall != 75 || r.Mock || r.Language != "zh" {
		t.Fatal(r)
	}
}
