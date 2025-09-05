package tools

import (
    "context"
    "encoding/base64"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    pdfx "github.com/ledongthuc/pdf"
)

type PDFExtractTool struct{}

func (t *PDFExtractTool) Name() string { return "pdf_extract" }

func (t *PDFExtractTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    dataB64, _ := inputs["data_base64"].(string)
    if dataB64 == "" {
        return nil, "", fmt.Errorf("missing data_base64")
    }
    // decode base64 (allow data: URIs)
    if i := strings.Index(dataB64, ","); i != -1 { dataB64 = dataB64[i+1:] }
    buf, err := base64.StdEncoding.DecodeString(dataB64)
    if err != nil { return nil, "", fmt.Errorf("invalid base64: %w", err) }
    // write to temp file because pdf lib expects a path
    dir := os.TempDir()
    path := filepath.Join(dir, fmt.Sprintf("pdf_%d.pdf", os.Getpid()))
    if err := os.WriteFile(path, buf, 0600); err != nil { return nil, "", err }
    defer os.Remove(path)
    // open and extract text
    f, r, err := pdfx.Open(path)
    if err != nil { return nil, "", err }
    defer f.Close()
    var out strings.Builder
    totalPages := r.NumPage()
    // optional: page ranges not implemented in MVP
    for page := 1; page <= totalPages; page++ {
        p := r.Page(page)
        txt, _ := p.GetPlainText(nil)
        if strings.TrimSpace(txt) != "" {
            out.WriteString(txt)
            out.WriteString("\n\n")
        }
    }
    text := strings.TrimSpace(out.String())
    return map[string]any{"text": text, "pages": totalPages}, fmt.Sprintf("pages=%d bytes=%d", totalPages, len(buf)), nil
}
