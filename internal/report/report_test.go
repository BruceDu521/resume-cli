package report

import (
	"strings"
	"testing"

	"resume-cli/internal/domain"
)

func TestLanguagesAndUnknown(t *testing.T) {
	a := domain.Assessment{Overall: 42, NotRequired: []string{"education"}, Findings: []domain.Finding{}}
	for _, s := range []string{"unknown", "partial", "unmet", "satisfied"} {
		a.Findings = append(a.Findings, domain.Finding{Requirement: domain.Requirement{Text: "Go"}, Judgment: domain.Judgment{Status: s}})
	}
	zh, en := Render(a, "zh", true), Render(a, "en", true)
	if zh.Overall != en.Overall || len(zh.Questions) != 3 || !strings.Contains(zh.Comment, "不代表") || !strings.Contains(en.Comment, "not proof") || !zh.Mock {
		t.Fatal(zh, en)
	}
	a.Findings = a.Findings[3:]
	if len(Render(a, "zh", false).Questions) != 1 || len(Render(a, "en", false).Questions) != 1 {
		t.Fatal("missing satisfied question")
	}
}

func TestEvidenceConflictIsVisibleInBothLanguages(t *testing.T) {
	a := domain.Assessment{Overall: 0, Findings: []domain.Finding{{
		Requirement: domain.Requirement{Text: "Go"},
		Judgment:    domain.Judgment{Status: "unknown", Score: 0, ReviewReason: "model_judgment_without_evidence"},
	}}}
	for lang, warning := range map[string]string{"zh": "需复核，暂不计分", "en": "needs review and earns no credit"} {
		r := Render(a, lang, false)
		if !strings.Contains(r.Comment, warning) || r.Overall != 0 || len(r.Questions) != 1 || r.Mock {
			t.Fatalf("%s hides evidence conflict: %+v", lang, r)
		}
	}
}
