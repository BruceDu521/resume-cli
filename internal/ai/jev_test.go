package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"resume-cli/internal/domain"
)

func TestJevBatchAndNoPII(t *testing.T) {
	c := domain.Candidate{Resume: domain.Resume{Email: "private@example.com"}, Facts: []domain.Fact{{ID: "fact", Category: "skill", BlockID: "b1", Quote: "Go"}}}
	job := domain.Job{Requirements: []domain.Requirement{{ID: "req", Category: "skill", Text: "Go", Required: true}}}
	j := Jev{Key: "synthetic", Model: "jev-1.13.0", BaseURL: "https://example.invalid/v1", HTTP: &Transport{Client: doFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		b, _ := json.Marshal(body)
		if strings.Contains(string(b), "private@example.com") {
			t.Fatal("unneeded PII")
		}
		if len(body["questions"].(map[string]any)) != 2 {
			t.Fatal("not batched")
		}
		return response(200, `{"model":"jev-1.13.0","answers":{"0s":{"type":"choice","choice":"satisfied","confidence":0.9,"probabilities":{"satisfied":0.9,"partial":0.1,"unmet":0,"unknown":0}},"0e":{"type":"choice","choice":"f0","confidence":1,"probabilities":{"none":0,"f0":1}}},"usage":{"input_tokens":100,"output_tokens":20}}`), nil
	})}}
	out, e := j.Match(context.Background(), c, job)
	if e != nil || len(out) != 1 || out[0].EvidenceID != "fact" || out[0].Score != 100 {
		t.Fatal(out, e)
	}
	if _, e = domain.Aggregate(c, job, out); e != nil {
		t.Fatal(e)
	}
	j.Key = ""
	if _, e = j.Match(context.Background(), c, job); e == nil {
		t.Fatal("missing key")
	}
}
func TestChoiceValidation(t *testing.T) {
	opts := map[string]any{"a": nil, "b": nil}
	for _, a := range []answer{{Type: "choice", Choice: "c", Probabilities: map[string]float64{"a": 1, "b": 0}}, {Type: "choice", Choice: "a", Probabilities: map[string]float64{"a": .1, "b": .9}}, {Type: "choice", Choice: "a", Probabilities: map[string]float64{"a": .9, "b": .9}}, {Type: "choice", Choice: "a", Confidence: 2, Probabilities: map[string]float64{"a": 1, "b": 0}}} {
		if validateChoice(a, opts) == nil {
			t.Fatal("invalid answer accepted")
		}
	}
}

func TestRoundedDistribution(t *testing.T) {
	// Observed API response: independently rounded probabilities sum to 1.01.
	opts := map[string]any{"a": nil, "b": nil, "c": nil, "d": nil}
	a := answer{Type: "choice", Choice: "a", Confidence: .41, Probabilities: map[string]float64{"a": .5700000000000001, "b": .41, "c": .02, "d": .01}}
	if err := validateChoice(a, opts); err != nil {
		t.Fatal(err)
	}
	a.Probabilities["a"] = .8
	if validateChoice(a, opts) == nil {
		t.Fatal("accepted materially inconsistent probabilities")
	}
}

func TestNoEvidenceCannotEarnCredit(t *testing.T) {
	c := domain.Candidate{Facts: []domain.Fact{{ID: "f", Category: "experience", Quote: "Developer 2018-2022"}}}
	job := domain.Job{Requirements: []domain.Requirement{{ID: "r", Category: "experience", Text: "Seven years", Required: true}}}
	j := Jev{Key: "synthetic", Model: "jev-1.13.0", BaseURL: "https://example.invalid", HTTP: &Transport{Client: doFunc(func(*http.Request) (*http.Response, error) {
		return response(200, `{"answers":{"0s":{"type":"choice","choice":"partial","confidence":0.4,"probabilities":{"satisfied":0,"partial":0.6,"unmet":0,"unknown":0.4}},"0e":{"type":"choice","choice":"none","confidence":0.7,"probabilities":{"f0":0.2,"none":0.8}}}}`), nil
	})}}
	out, err := j.Match(context.Background(), c, job)
	if err != nil || len(out) != 1 || out[0].Status != "unknown" || out[0].Score != 0 || out[0].ReviewReason == "" {
		t.Fatal(out, err)
	}
	if _, err = domain.Aggregate(c, job, out); err != nil {
		t.Fatal(err)
	}
}
