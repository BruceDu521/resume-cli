// Package samples contains synthetic demo inputs shipped inside the binary.
package samples

import "embed"

// Files contains only public, synthetic fixtures; never embed candidate documents.
//
//go:embed testdata/resume-zh.pdf testdata/resume-en.pdf testdata/jd.txt testdata/jd-en.txt
var Files embed.FS
