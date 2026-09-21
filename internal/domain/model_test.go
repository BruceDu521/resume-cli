package domain

import (
	"math"
	"testing"
)

func fixture() (Candidate, Job, []Judgment) {
	return Candidate{Facts: []Fact{{ID: "f1"}, {ID: "f2"}}}, Job{Requirements: []Requirement{{ID: "r1", Category: "skill", Required: true}, {ID: "r2", Category: "experience", Required: true}}}, []Judgment{{RequirementID: "r1", Status: "satisfied", Score: 100, EvidenceID: "f1", Confidence: .9}, {RequirementID: "r2", Status: "partial", Score: 50, EvidenceID: "f2", Confidence: .8}}
}
func TestAggregateWeights(t *testing.T) {
	c, j, v := fixture()
	a, e := Aggregate(c, j, v)
	if e != nil {
		t.Fatal(e)
	}
	if a.Overall != 79 || a.Skill != 100 || a.Experience != 50 || a.Education != 100 || len(a.NotRequired) != 1 {
		t.Fatalf("%+v", a)
	}
	v[1].Status = "unknown"
	v[1].EvidenceID = ""
	v[1].Score = 99
	a, e = Aggregate(c, j, v)
	if e != nil || a.Experience != 0 || a.Overall != 59 {
		t.Fatalf("unknown earned credit: %+v %v", a, e)
	}
}
func TestAggregateRejectsInvalid(t *testing.T) {
	for _, name := range []string{"missing", "duplicate", "badid", "status", "evidence", "unknownEvidence", "nan", "infinite", "negative", "overscore", "confidence"} {
		t.Run(name, func(t *testing.T) {
			c, j, v := fixture()
			switch name {
			case "missing":
				v = v[:1]
			case "duplicate":
				v[1] = v[0]
			case "badid":
				v[1].RequirementID = "no"
			case "status":
				v[1].Status = "great"
			case "evidence":
				v[1].EvidenceID = "no"
			case "unknownEvidence":
				v[1].Status = "unknown"
			case "nan":
				v[1].Score = math.NaN()
			case "infinite":
				v[1].Score = math.Inf(1)
			case "negative":
				v[1].Score = -1
			case "overscore":
				v[1].Score = 101
			case "confidence":
				v[1].Confidence = math.NaN()
			}
			if _, e := Aggregate(c, j, v); e == nil {
				t.Fatal("accepted")
			}
		})
	}
}
func TestRequiredWeight(t *testing.T) {
	c, _, _ := fixture()
	j := Job{Requirements: []Requirement{{ID: "a", Category: "skill", Required: true}, {ID: "b", Category: "skill"}}}
	v := []Judgment{{RequirementID: "a", Status: "satisfied", Score: 100, EvidenceID: "f1"}, {RequirementID: "b", Status: "unmet", Score: 99, EvidenceID: "f2"}}
	a, e := Aggregate(c, j, v)
	if e != nil || a.Skill != 67 || a.Overall != 67 {
		t.Fatalf("%+v %v", a, e)
	}
}
func TestCandidateEvidence(t *testing.T) {
	d := NewDocument("林予安\nGo 开发\f本科")
	c := Candidate{Resume: Resume{Name: "林予安", Education: []Education{}, Skills: []string{"Go"}}, Facts: []Fact{{ID: "f1", Category: "skill", BlockID: "b2", Quote: "Go 开发"}}}
	if e := c.Validate(d); e != nil {
		t.Fatal(e)
	}
	if d.Blocks[2].Page != 2 {
		t.Fatal("page lost")
	}
	c.Facts[0].Quote = "Rust"
	if c.Validate(d) == nil {
		t.Fatal("invented evidence")
	}
	c.Facts = []Fact{}
	c.Resume.Name = "别的人"
	if c.Validate(d) != nil {
		t.Fatal("profile wording must not block scoring")
	}
}
func TestJobValidation(t *testing.T) {
	for _, j := range []Job{{}, {Requirements: []Requirement{{ID: "x", Category: "age", Text: "Go"}}}, {Requirements: []Requirement{{ID: "x", Category: "skill", Text: "Rust"}}}, {Requirements: []Requirement{{ID: "x", Category: "skill", Text: "Go"}, {ID: "x", Category: "skill", Text: "Go"}}}} {
		if j.Validate("Go") == nil {
			t.Fatalf("accepted %+v", j)
		}
	}
}

func TestGroundPreservesContextAndMissingProfileEvidence(t *testing.T) {
	d := NewDocument("Alice\nNo Rust production experience.\nGo / PostgreSQL backend development\nSample University | Bachelor | 2020")
	c := Candidate{Resume: Resume{Name: "Alice", Skills: []string{"Go", "PostgreSQL"}, Education: []Education{{School: "Sample University", Degree: "Bachelor", GraduationTime: "2020"}}}, Facts: []Fact{{ID: "f1", Category: "skill", BlockID: "b2", Quote: "Rust"}}}
	got, err := c.Ground(d)
	if err != nil || len(got.Facts) != 3 {
		t.Fatal(got, err)
	}
	if got.Facts[0].Quote != "No Rust production experience." || got.Facts[0].ID != "f1" {
		t.Fatal("negation lost", got)
	}
	if c.Facts[0].Quote != "Rust" {
		t.Fatal("mutated caller's evidence")
	}
	again, err := got.Ground(d)
	if err != nil || len(again.Facts) != 3 {
		t.Fatal("grounding not idempotent", again, err)
	}
	c.Facts = []Fact{}
	got, err = c.Ground(d)
	if err != nil || len(got.Facts) != 2 {
		t.Fatal("profile evidence omitted", got, err)
	}
	c.Resume.Skills = []string{"invented skill"}
	if got, err = c.Ground(d); err != nil || len(got.Facts) != 1 {
		t.Fatal("profile text must not fabricate evidence", got, err)
	}
}
