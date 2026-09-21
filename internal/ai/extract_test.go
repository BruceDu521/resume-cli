package ai

import (
	"context"
	"encoding/json"
	"resume-cli/internal/domain"
	"strings"
	"testing"
)

type extractionCapture struct {
	request Request
	data    []byte
}

func (g *extractionCapture) Identity() string { return "offline" }
func (g *extractionCapture) Generate(_ context.Context, r Request) ([]byte, Usage, error) {
	g.request = r
	return g.data, Usage{Stage: r.Stage}, nil
}
func TestPublicExtractUsesFullTextAndOnlyPublicSchema(t *testing.T) {
	text := "Alice\nImplemented backends in golang.\nBuilt release automation for a cluster."
	want := domain.Resume{Name: "Alice", Education: []domain.Education{}, Skills: []string{"Go", "Release automation"}}
	data, _ := json.Marshal(want)
	g := &extractionCapture{data: data}
	got, err := (Structurer{Generator: g}).Extract(context.Background(), domain.NewDocument(text))
	if err != nil || got.Skills[0] != "Go" {
		t.Fatal(got, err)
	}
	if g.request.State != text || g.request.Stage != "extract" {
		t.Fatal("did not pass full original text")
	}
	schema, _ := json.Marshal(g.request.Schema)
	for _, forbidden := range []string{"facts", "block_id", "end_block_id"} {
		if strings.Contains(string(schema), forbidden) {
			t.Fatal("evidence leaked into public extraction contract")
		}
	}
	if strings.Contains(g.request.Instruction, "64") {
		t.Fatal("arbitrary evidence limit in extraction")
	}
}
func TestPublicExtractRejectsInvalidShape(t *testing.T) {
	for _, raw := range []string{
		`{"name":"Alice"}`,
		`{"name":"Alice","phone":"","email":"","city":"","education":[],"skills":[],"facts":[]}`,
		`{"name":"Alice","phone":"","email":"","city":"","education":[],"skills":[""]}`,
	} {
		g := &extractionCapture{data: []byte(raw)}
		if _, err := (Structurer{Generator: g}).Extract(context.Background(), domain.NewDocument("Alice")); err == nil {
			t.Fatal("invalid public shape accepted")
		}
	}
}
