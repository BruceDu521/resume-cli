package jsonutil

import (
	"encoding/json"
	"strings"
	"testing"
)

type sample struct {
	Name  string   `json:"name"`
	Items []string `json:"items"`
	Flag  bool     `json:"flag"`
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name, input  string
		ok, repaired bool
	}{
		{"valid", `{"name":"林予安","items":[],"flag":false}`, true, false},
		{"fence", "```json\n{\"name\":\"a\",\"items\":[],\"flag\":false}\n```", true, true},
		{"trailing", `{"name":"comma,} and quote \"","items":["x",],"flag":true,}`, true, true},
		{"missing", `{"name":"a","items":[]}`, false, false},
		{"duplicate", `{"name":"a","name":"b","items":[],"flag":true}`, false, false},
		{"nestedDuplicate", `{"name":"a","items":[],"flag":true,"unknown":{"x":1,"x":2}}`, false, false},
		{"unknown", `{"name":"a","items":[],"flag":true,"extra":1}`, false, false},
		{"null", `{"name":null,"items":[],"flag":true}`, false, false},
		{"type", `{"name":17,"items":[],"flag":true}`, false, false},
		{"truncated", `{"name":"a"`, false, false},
		{"multiple", `{} {}`, false, false},
		{"prose", `Here is {"name":"a"}`, false, false},
		{"quotes", `{'name':'a'}`, false, false},
		{"oversize", strings.Repeat(" ", MaxBytes+1), false, false},
		{"array", `[]`, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v sample
			r, e := Decode([]byte(tt.input), &v)
			if (e == nil) != tt.ok {
				t.Fatalf("%v", e)
			}
			if tt.ok && r != tt.repaired {
				t.Fatal("wrong repair flag")
			}
		})
	}
}
func FuzzDecode(f *testing.F) {
	for _, s := range []string{`{"name":"a","items":[],"flag":false}`, "```json\n{}\n```", `{"a":",}"}`, `{"a":"\\\""}`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		var v sample
		_, e := Decode([]byte(s), &v)
		if e == nil {
			b, e := json.Marshal(v)
			if e != nil || !json.Valid(b) {
				t.Fatal("invalid accepted result")
			}
		}
	})
}
