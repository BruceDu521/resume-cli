// Package domain contains provider-independent data and scoring rules.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"
)

const PolicyVersion = "evidence-v1"

type Block struct {
	ID   string `json:"id"`
	Page int    `json:"page"`
	Text string `json:"text"`
}
type Document struct {
	Text   string  `json:"text"`
	Hash   string  `json:"hash"`
	Blocks []Block `json:"blocks"`
}

func Digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func NewDocument(text string) Document {
	d := Document{Text: text, Hash: Digest(text), Blocks: []Block{}}
	for p, page := range strings.Split(text, "\f") {
		for _, line := range strings.Split(page, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				d.Blocks = append(d.Blocks, Block{fmt.Sprintf("b%d", len(d.Blocks)+1), p + 1, line})
			}
		}
	}
	return d
}

type Education struct {
	School         string `json:"school"`
	Major          string `json:"major"`
	Degree         string `json:"degree"`
	GraduationTime string `json:"graduation_time"`
}
type Resume struct {
	Name      string      `json:"name"`
	Phone     string      `json:"phone"`
	Email     string      `json:"email"`
	City      string      `json:"city"`
	Education []Education `json:"education"`
	Skills    []string    `json:"skills"`
}

// Validate checks the public extraction contract, not semantic equivalence.
// Skills may be normalized or inferred from described work; exact substring
// matching is inappropriate for this public extraction task.
func (r Resume) Validate() error {
	if r.Education == nil || r.Skills == nil {
		return errors.New("education and skills must be arrays")
	}
	for _, skill := range r.Skills {
		if strings.TrimSpace(skill) == "" {
			return errors.New("skills must not contain empty entries")
		}
	}
	return nil
}

type Fact struct {
	ID         string `json:"id"`
	Category   string `json:"category"`
	BlockID    string `json:"block_id"`
	EndBlockID string `json:"end_block_id,omitempty"`
	Quote      string `json:"quote"`
}
type Candidate struct {
	Resume Resume `json:"resume"`
	Facts  []Fact `json:"facts"`
}
type Requirement struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Text     string `json:"text"`
	Required bool   `json:"required"`
}
type Job struct {
	Requirements []Requirement `json:"requirements"`
}
type Judgment struct {
	RequirementID string   `json:"requirement_id"`
	Status        string   `json:"status"`
	Score         float64  `json:"score"`
	EvidenceID    string   `json:"evidence_id,omitempty"`
	EvidenceIDs   []string `json:"evidence_ids,omitempty"`
	Confidence    float64  `json:"confidence,omitempty"`
	ReviewReason  string   `json:"review_reason,omitempty"`
}
type Finding struct {
	Requirement Requirement `json:"requirement"`
	Judgment    Judgment    `json:"judgment"`
	Evidence    *Fact       `json:"evidence,omitempty"`
	Evidences   []Fact      `json:"evidences,omitempty"`
}
type Assessment struct {
	Overall     int       `json:"overall_score"`
	Skill       int       `json:"skill_score"`
	Experience  int       `json:"experience_score"`
	Education   int       `json:"education_score"`
	NotRequired []string  `json:"not_required,omitempty"`
	Findings    []Finding `json:"findings,omitempty"`
	Policy      string    `json:"policy_version"`
}

func category(s string) bool { return s == "skill" || s == "experience" || s == "education" }

// Compact allows harmless whitespace differences, not changes to source facts.
func Compact(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}
func Contains(source, quote string) bool {
	q := Compact(quote)
	return q != "" && strings.Contains(Compact(source), q)
}
func (c Candidate) Validate(d Document) error {
	if c.Resume.Education == nil || c.Resume.Skills == nil || c.Facts == nil {
		return errors.New("candidate collections must be arrays")
	}
	if err := c.Resume.Validate(); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, f := range c.Facts {
		span, err := d.EvidenceText(f)
		if f.ID == "" || seen[f.ID] || !category(f.Category) {
			return errors.New("invalid candidate evidence identity or category")
		}
		if err != nil {
			return err
		}
		if !Contains(span, f.Quote) {
			return errors.New("candidate quote does not match its declared source range")
		}
		seen[f.ID] = true
	}
	return nil
}

// EvidenceText resolves only an explicit contiguous source range, never searches
// the whole document for a replacement location. Page boundaries remain visible.
func (d Document) EvidenceText(f Fact) (string, error) {
	start, end := -1, -1
	endID := f.EndBlockID
	if endID == "" {
		endID = f.BlockID
	}
	for i, b := range d.Blocks {
		if b.ID == f.BlockID {
			start = i
		}
		if b.ID == endID {
			end = i
		}
	}
	if start < 0 || end < start {
		return "", errors.New("candidate evidence requires a valid ordered source range")
	}
	lines := make([]string, 0, end-start+1)
	for _, b := range d.Blocks[start : end+1] {
		lines = append(lines, b.Text)
	}
	return strings.Join(lines, "\n"), nil
}

// Ground preserves the source context of selected excerpts and supplies source
// blocks for extracted profile fields that would otherwise be lost to matching.
// It copies text already present in the document; it does not infer qualifications.
func (c Candidate) Ground(d Document) (Candidate, error) {
	if err := c.Validate(d); err != nil {
		return c, err
	}
	c.Facts = append([]Fact{}, c.Facts...)
	ids := map[string]bool{}
	for i, f := range c.Facts {
		c.Facts[i].Quote, _ = d.EvidenceText(f)
		ids[f.ID] = true
	}
	type field struct{ value, category string }
	fields := []field{}
	for _, skill := range c.Resume.Skills {
		fields = append(fields, field{skill, "skill"})
	}
	for _, edu := range c.Resume.Education {
		for _, value := range []string{edu.School, edu.Major, edu.Degree, edu.GraduationTime} {
			fields = append(fields, field{value, "education"})
		}
	}
	for _, v := range fields {
		if v.value == "" {
			continue
		}
		covered := false
		for _, f := range c.Facts {
			if Contains(f.Quote, v.value) {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		for _, b := range d.Blocks {
			if !Contains(b.Text, v.value) {
				continue
			}
			id := "source_" + b.ID
			for ids[id] {
				id += "_"
			}
			ids[id] = true
			c.Facts = append(c.Facts, Fact{ID: id, Category: v.category, BlockID: b.ID, Quote: b.Text})
			break
		}
	}
	return c, c.Validate(d)
}
func (j Job) Validate(text string) error {
	if len(j.Requirements) == 0 {
		return errors.New("JD must contain at least one assessable requirement")
	}
	seen := map[string]bool{}
	for _, r := range j.Requirements {
		if r.ID == "" || seen[r.ID] || !category(r.Category) || !Contains(text, r.Text) {
			return errors.New("invalid or unsupported JD requirement")
		}
		seen[r.ID] = true
	}
	return nil
}

// Aggregate measures demonstrated evidence, not a person's unobserved ability.
// Unknown criteria earn no demonstrated credit; absent dimensions are excluded.
func Aggregate(c Candidate, j Job, judgments []Judgment) (Assessment, error) {
	a := Assessment{NotRequired: []string{}, Findings: []Finding{}, Policy: PolicyVersion}
	if len(j.Requirements) == 0 || len(judgments) != len(j.Requirements) {
		return a, errors.New("incomplete assessment")
	}
	facts := map[string]Fact{}
	for _, f := range c.Facts {
		facts[f.ID] = f
	}
	byID := map[string]Judgment{}
	for _, v := range judgments {
		if _, ok := byID[v.RequirementID]; ok {
			return a, errors.New("duplicate judgment")
		}
		byID[v.RequirementID] = v
	}
	sums, weights := map[string]float64{}, map[string]float64{}
	for _, r := range j.Requirements {
		v, ok := byID[r.ID]
		if !ok || !category(r.Category) || math.IsNaN(v.Score) || math.IsInf(v.Score, 0) || v.Score < 0 || v.Score > 100 || math.IsNaN(v.Confidence) || v.Confidence < 0 || v.Confidence > 1 {
			return a, errors.New("invalid judgment")
		}
		if v.Status != "satisfied" && v.Status != "partial" && v.Status != "unmet" && v.Status != "unknown" {
			return a, errors.New("invalid evidence status")
		}
		ids := v.EvidenceIDs
		if v.EvidenceID != "" {
			if len(ids) > 0 {
				return a, errors.New("use evidence_ids or legacy evidence_id, not both")
			}
			ids = []string{v.EvidenceID}
		}
		if v.Status != "unknown" && len(ids) == 0 {
			return a, errors.New("judgment requires source evidence")
		}
		if v.Status == "unknown" && len(ids) > 0 {
			return a, errors.New("unknown judgment must not claim evidence")
		}
		evidence := []Fact{}
		seenEvidence := map[string]bool{}
		for _, id := range ids {
			f, ok := facts[id]
			if !ok || seenEvidence[id] {
				return a, errors.New("invalid or duplicate judgment evidence reference")
			}
			seenEvidence[id] = true
			evidence = append(evidence, f)
		}
		// Do not let numerical uncertainty give unsupported claims positive credit.
		if v.Status == "unknown" || v.Status == "unmet" {
			v.Score = 0
		}
		w := 1.0
		if r.Required {
			w = 2
		}
		sums[r.Category] += v.Score * w
		weights[r.Category] += w
		finding := Finding{Requirement: r, Judgment: v}
		if v.EvidenceID != "" {
			copy := evidence[0]
			finding.Evidence = &copy
		} else if len(evidence) > 0 {
			finding.Evidences = evidence
		}
		a.Findings = append(a.Findings, finding)
	}
	total, denom := 0.0, 0.0
	for _, dim := range []struct {
		name   string
		weight float64
		dst    *int
	}{{"skill", .5, &a.Skill}, {"experience", .35, &a.Experience}, {"education", .15, &a.Education}} {
		if weights[dim.name] == 0 {
			*dim.dst = 100
			a.NotRequired = append(a.NotRequired, dim.name)
			continue
		}
		score := sums[dim.name] / weights[dim.name]
		*dim.dst = int(math.Round(score))
		total += score * dim.weight
		denom += dim.weight
	}
	if denom == 0 {
		return a, errors.New("JD has no applicable dimensions")
	}
	a.Overall = int(math.Round(total / denom))
	return a, nil
}
