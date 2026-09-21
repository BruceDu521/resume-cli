package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"resume-cli/internal/domain"
	"strings"
	"testing"
)

func TestSingleScoreKeepsManyRequirementsAndDistributedEvidence(t *testing.T) {
	d := domain.NewDocument("Project A: built APIs using golang.\nUnrelated job\nProject B: operated PostgreSQL backups.")
	v := evaluation{Matches: []match{}, Comment: "Both projects provide relevant evidence.", Questions: []string{"Explain how the systems interacted?"}}
	requirements := []string{}
	for i := 0; i < 30; i++ {
		text := fmt.Sprintf("Requirement %d: Go and PostgreSQL", i+1)
		requirements = append(requirements, text)
		v.Matches = append(v.Matches, match{Requirement: text, Category: "skill", Required: true, Status: "satisfied", Evidence: []citation{
			{BlockID: "b1", Quote: "built APIs using golang"},
			{BlockID: "b3", Quote: "operated PostgreSQL backups"},
		}})
	}
	g := &sequenceGenerator{}
	// Capture the actual schema and number of requests: no correction should be
	// triggered by golang -> Go reasoning, 30 requirements or disjoint citations.
	data := mustJSON(t, v)
	g.bodies = []string{data}
	c, j, judgments, _, _, err := (Structurer{Generator: g}).Evaluate(context.Background(), d, strings.Join(requirements, "\n"), "en")
	if err != nil || g.calls != 1 || len(j.Requirements) != 30 || len(judgments) != 30 {
		t.Fatal(len(j.Requirements), g.calls, err)
	}
	props := g.requests[0].Schema["properties"].(map[string]any)
	if len(props) != 3 || props["matches"] == nil || props["candidate"] != nil || props["resume"] != nil {
		t.Fatal("unnecessary model output", props)
	}
	if c.Resume.Name != "" || c.Resume.Phone != "" {
		t.Fatal("single scoring depends on profile extraction")
	}
	a, err := domain.Aggregate(c, j, judgments)
	if err != nil || len(a.Findings) != 30 || len(a.Findings[0].Evidences) != 2 {
		t.Fatal(a, err)
	}
	if a.Findings[0].Evidences[0].BlockID != "b1" || a.Findings[0].Evidences[1].BlockID != "b3" {
		t.Fatal("disjoint evidence lost")
	}
	v.Matches[0].Evidence[1].Quote = "invented backup experience"
	if _, _, _, err := v.assessment(d, strings.Join(requirements, "\n")); err == nil {
		t.Fatal("invented quotation accepted")
	}
}

func TestSingleScoreAllowsAllRequirementsUnknown(t *testing.T) {
	d := domain.NewDocument("No relevant qualifications described")
	v := evaluation{Matches: []match{{Requirement: "Rust", Category: "skill", Required: true, Status: "unknown", Evidence: []citation{}}}, Comment: "Resume does not establish Rust.", Questions: []string{"Have you used Rust?"}}
	c, j, judgments, _, _, err := (Structurer{Generator: fakeGenerator{value: v}}).Evaluate(context.Background(), d, "Rust", "en")
	if err != nil {
		t.Fatal(err)
	}
	a, err := domain.Aggregate(c, j, judgments)
	if err != nil || a.Overall != 0 {
		t.Fatal(a, err)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
