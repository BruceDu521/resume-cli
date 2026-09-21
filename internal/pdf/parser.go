// Package pdf adapts the local Poppler executable without shell interpolation.
package pdf

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"resume-cli/internal/domain"
	"resume-cli/internal/fileio"

	"resume-cli/internal/i18n"
)

// Image-heavy resumes can be large; extracted text has a separate AI-input limit.
const DefaultPDFBytes = 32 << 20
const DefaultTextBytes = 64 << 10
const MaxPDFBytes = 200 << 20
const MaxTextBytes = 256 << 10

type Parser struct {
	Binary string
	// Zero uses the default; negative limits are invalid.
	MaxPDFBytes  int64
	MaxTextBytes int
}
type bounded struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (b *bounded) Len() int       { return b.buffer.Len() }
func (b *bounded) String() string { return b.buffer.String() }

func (b *bounded) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		b.exceeded = true
		return 0, errors.New("PDF text exceeds limit")
	}
	return b.buffer.Write(p)
}
func (p Parser) Parse(ctx context.Context, path string) (doc domain.Document, resultErr error) {
	defer func() {
		if resultErr != nil {
			var inputErr *fileio.Error
			if !errors.As(resultErr, &inputErr) {
				message := resultErr.Error()
				if errors.Is(resultErr, context.Canceled) {
					message = "解析已取消。"
				}
				if errors.Is(resultErr, context.DeadlineExceeded) {
					message = "PDF 解析超时，请检查文件，或用 --timeout 2m 延长等待时间。"
				}
				resultErr = &fileio.Error{Path: path, Message: message, Cause: resultErr}
			}
		}
	}()
	if err := ctx.Err(); err != nil {
		return doc, err
	}
	pdfLimit, textLimit := p.MaxPDFBytes, p.MaxTextBytes
	if pdfLimit == 0 {
		pdfLimit = DefaultPDFBytes
	}
	if textLimit == 0 {
		textLimit = DefaultTextBytes
	}
	if pdfLimit < 0 || textLimit < 0 || pdfLimit > int64(^uint(0)>>1)-1 {
		return doc, i18n.New("资源上限必须是正数，且不能超过本机可表示的字节范围。")
	}
	if pdfLimit > MaxPDFBytes || textLimit > MaxTextBytes {
		return doc, i18n.New("PDF 文件上限不能超过 200 MiB，提取文本上限不能超过 256 KiB。")
	}
	data, err := fileio.Read(path, pdfLimit)
	if err != nil {
		return domain.Document{}, err
	}
	if len(data) == 0 {
		return doc, i18n.New("文件为空，请提供包含文字内容的 PDF 简历。")
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return domain.Document{}, i18n.New("文件不是 PDF，请提供有效的 PDF 简历；修改扩展名不能转换格式。")
	}
	// Copy bounded bytes into a private file, avoiding path/option injection and TOCTOU.
	f, err := os.CreateTemp("", "resume-cli-*.pdf")
	if err != nil {
		return doc, &fileio.Error{Path: path, Message: "无法创建解析所需的临时文件，请检查临时目录权限和磁盘空间。", Cause: err}
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return doc, &fileio.Error{Path: path, Message: "无法写入解析所需的临时文件，请检查磁盘空间。", Cause: err}
	}
	if err = f.Close(); err != nil {
		return doc, &fileio.Error{Path: path, Message: "无法保存解析所需的临时文件，请检查磁盘空间。", Cause: err}
	}
	bin := p.Binary
	if bin == "" {
		bin = "pdftotext"
	}
	cmd := exec.CommandContext(ctx, bin, "-enc", "UTF-8", f.Name(), "-")
	output, stderr := &bounded{limit: textLimit}, &bounded{limit: 16 << 10}
	cmd.Stdout = output
	cmd.Stderr = stderr
	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return domain.Document{}, ctx.Err()
		}
		var ex *exec.Error
		if errors.As(err, &ex) || errors.Is(err, os.ErrNotExist) {
			return domain.Document{}, i18n.New("找不到 PDF 解析工具 pdftotext。macOS 请运行 brew install poppler；Debian/Ubuntu 请安装 poppler-utils 和 poppler-data。")
		}
		if output.exceeded {
			return doc, &fileio.Error{Path: path, Message: "PDF 提取文本超过 %d 字节上限；请减少内容或调大 --max-text-kib 后重试。", Args: []any{textLimit}}
		}
		reason := "PDF 无法解析，可能已损坏或格式不受支持；请确认能正常打开，并重新导出 PDF。"
		if strings.Contains(strings.ToLower(stderr.String()), "password") || strings.Contains(strings.ToLower(stderr.String()), "encrypted") {
			reason = "PDF 已加密或需要密码，请先解除保护并导出可读取的 PDF。"
		}
		return doc, &fileio.Error{Path: path, Message: reason, Cause: err}
	}
	text := output.String()
	if strings.Contains(stderr.String(), "Missing language pack") {
		return domain.Document{}, i18n.New("缺少 PDF 字符映射数据，请安装 poppler-data 后重试。")
	}
	if !utf8.ValidString(text) {
		return domain.Document{}, i18n.New("PDF 提取文本编码异常，请重新导出 PDF 后重试。")
	}
	if strings.TrimSpace(text) == "" {
		return domain.Document{}, i18n.New("PDF 中没有可提取的文字；若为扫描件或图片，请先进行 OCR 文字识别后重试（本工具不含 OCR）。")
	}
	return domain.NewDocument(text), nil
}
