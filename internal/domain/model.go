// Package domain contains provider-independent document and resume data.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

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
