// Package pdf adapts the local Poppler executable without shell interpolation.
package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"resume-cli/internal/domain"
	"resume-cli/internal/fileio"
)

const MaxPDFBytes = 20 << 20
const MaxTextBytes = 160 << 10

type Parser struct{ Binary string }
type bounded struct {
	bytes.Buffer
	limit int
}

func (b *bounded) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		return 0, errors.New("PDF text exceeds limit")
	}
	return b.Buffer.Write(p)
}
func (p Parser) Parse(ctx context.Context, path string) (domain.Document, error) {
	data, err := fileio.Read(path, MaxPDFBytes)
	if err != nil {
		return domain.Document{}, err
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return domain.Document{}, errors.New("input is not a PDF")
	}
	// Copy bounded bytes into a private file, avoiding path/option injection and TOCTOU.
	f, err := os.CreateTemp("", "resume-cli-*.pdf")
	if err != nil {
		return domain.Document{}, err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return domain.Document{}, err
	}
	if err = f.Close(); err != nil {
		return domain.Document{}, err
	}
	bin := p.Binary
	if bin == "" {
		bin = "pdftotext"
	}
	cmd := exec.CommandContext(ctx, bin, "-enc", "UTF-8", f.Name(), "-")
	output, stderr := &bounded{limit: MaxTextBytes}, &bounded{limit: 16 << 10}
	cmd.Stdout = output
	cmd.Stderr = stderr
	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return domain.Document{}, ctx.Err()
		}
		var ex *exec.Error
		if errors.As(err, &ex) {
			return domain.Document{}, errors.New("Poppler pdftotext is unavailable; install poppler")
		}
		return domain.Document{}, fmt.Errorf("PDF cannot be read, is encrypted, or exceeds text limits: %w", err)
	}
	text := output.String()
	if strings.Contains(stderr.String(), "Missing language pack") {
		return domain.Document{}, errors.New("Poppler language mappings are missing; install poppler-data")
	}
	if !utf8.ValidString(text) {
		return domain.Document{}, errors.New("PDF text is not valid UTF-8")
	}
	if strings.TrimSpace(text) == "" {
		return domain.Document{}, errors.New("PDF text is empty; scanned documents require OCR (not supported)")
	}
	return domain.NewDocument(text), nil
}
