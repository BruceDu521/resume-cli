package domain

import (
	"strings"
	"testing"
)

func TestExplicitWrappedEvidence(t *testing.T) {
	d := NewDocument("Alice\nBuilt Kubernetes deployment and\nincident recovery tooling.\nNo Rust\nproduction experience.\fAnother project")
	for _, tt := range []struct {
		name, start, end, quote string
		ok                      bool
	}{
		{"wrapped", "b2", "b3", "Kubernetes deployment and incident recovery", true},
		{"single", "b2", "", "Kubernetes", true},
		{"negation", "b4", "b5", "No Rust production experience.", true},
		{"missing end", "b2", "", "Kubernetes deployment and incident recovery", false},
		{"wrong start", "b3", "b3", "Built Kubernetes", false},
		{"nonexistent", "b2", "b99", "Kubernetes", false},
		{"reverse", "b3", "b2", "Kubernetes", false},
		{"skip lines", "b2", "b5", "deployment and production experience", false},
		{"invented", "b2", "b3", "Kubernetes production expert", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := Candidate{Resume: Resume{Education: []Education{}, Skills: []string{}}, Facts: []Fact{{ID: "f", Category: "experience", BlockID: tt.start, EndBlockID: tt.end, Quote: tt.quote}}}
			err := c.Validate(d)
			if (err == nil) != tt.ok {
				t.Fatal(err)
			}
			if tt.ok {
				got, err := c.Ground(d)
				if err != nil {
					t.Fatal(err)
				}
				again, err := got.Ground(d)
				if err != nil || again.Facts[0] != got.Facts[0] {
					t.Fatal("unstable grounding", err)
				}
				if tt.name == "negation" && !strings.HasPrefix(got.Facts[0].Quote, "No Rust") {
					t.Fatal("negation lost")
				}
				if c.Facts[0].Quote != tt.quote {
					t.Fatal("mutated caller")
				}
			}
		})
	}
	d = NewDocument(strings.Repeat("line\n", 17))
	if _, err := d.EvidenceText(Fact{BlockID: "b1", EndBlockID: "b17"}); err == nil {
		t.Fatal("unbounded evidence")
	}
}

func TestSourceFieldErrorsDoNotRevealValues(t *testing.T) {
	c := Candidate{Resume: Resume{Name: "sensitive-name", Education: []Education{}, Skills: []string{}}, Facts: []Fact{}}
	err := c.Validate(NewDocument("Alice"))
	if err == nil || !strings.Contains(err.Error(), "resume.name") || strings.Contains(err.Error(), "sensitive-name") {
		t.Fatal(err)
	}
	c.Resume.Name = "Alice"
	c.Resume.Education = []Education{{GraduationTime: "2099-01"}}
	err = c.Validate(NewDocument("Alice\n2099.01"))
	if err == nil || !strings.Contains(err.Error(), "education[0].graduation_time") || strings.Contains(err.Error(), "2099") {
		t.Fatal(err)
	}
}
