package domain

import "testing"

func TestDocumentPreservesInputAndPageBoundaries(t *testing.T) {
	text := " Alice \n\nGo\fPostgreSQL\n"
	d := NewDocument(text)
	if d.Text != text || d.Hash != Digest(text) || len(d.Blocks) != 3 || d.Blocks[2].Page != 2 {
		t.Fatal(d)
	}
	if d.Hash == NewDocument(text+" ").Hash {
		t.Fatal("distinct input shares cache identity")
	}
}
func TestResumeCollections(t *testing.T) {
	for _, r := range []Resume{{}, {Education: []Education{}, Skills: []string{" "}}} {
		if r.Validate() == nil {
			t.Fatal("invalid collection accepted")
		}
	}
	if err := (Resume{Education: []Education{}, Skills: []string{"Go"}}).Validate(); err != nil {
		t.Fatal(err)
	}
}
