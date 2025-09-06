package tools

import (
    "context"
    "encoding/base64"
    "errors"
    "fmt"
    "path/filepath"
    "strings"
)

// FileExtractTool converts common file types into text for downstream LLM use.
// Inputs:
// - data_base64: string (required) — may be a data: URL
// - filename: string (optional)
// - content_type: string (optional)
// - max_bytes: number (optional) — override size limit (default 20MB via env FILE_MAX_BYTES)
// Output: string (extracted/plain text)
type FileExtractTool struct{}

func (t *FileExtractTool) Name() string { return "file_extract" }

func (t *FileExtractTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    b64, _ := inputs["data_base64"].(string)
    if b64 == "" { return nil, "", fmt.Errorf("missing data_base64") }
    if i := strings.Index(b64, ","); i != -1 { b64 = b64[i+1:] } // strip data: prefix
    buf, err := base64.StdEncoding.DecodeString(b64)
    if err != nil { return nil, "", fmt.Errorf("invalid base64: %w", err) }
    max := getInt(inputs, "max_bytes", envInt("FILE_MAX_BYTES", 20*1024*1024))
    if len(buf) > max { return nil, "", fmt.Errorf("file too large: %d bytes > limit %d", len(buf), max) }

    // Determine type
    filename, _ := inputs["filename"].(string)
    ctype, _ := inputs["content_type"].(string)
    ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
    // magic checks
    isPDF := strings.HasPrefix(string(buf), "%PDF-") || ext == "pdf" || strings.Contains(ctype, "pdf")
    if isPDF {
        // Reuse existing PDF tool for streaming support
        out, logs, err := (&PDFExtractTool{}).Execute(ctx, map[string]any{"data_base64": inputs["data_base64"], "max_bytes": max})
        return out, prependLog("pdf", logs), err
    }

    // Try HTML if bytes look like HTML or ext suggests so
    looksHTML := ext == "html" || ext == "htm" || strings.Contains(ctype, "html")
    if !looksHTML {
        s := strings.ToLower(string(buf))
        if strings.Contains(s, "<html") || strings.Contains(s, "<body") { looksHTML = true }
    }
    if looksHTML {
        out, logs, err := (&HTMLToTextTool{}).Execute(ctx, map[string]any{"html": string(buf)})
        return out, prependLog("html", logs), err
    }

    // Plain-text-ish types: txt, md, markdown, csv, json, log, yaml, yml
    if ext == "txt" || ext == "md" || ext == "markdown" || ext == "csv" || ext == "json" || ext == "log" || ext == "yaml" || ext == "yml" ||
        strings.Contains(ctype, "text/") || strings.Contains(ctype, "json") || strings.Contains(ctype, "csv") || strings.Contains(ctype, "yaml") {
        // Return as string (no parsing); callers can choose summarize or regex/csv tools next
        text := strings.TrimSpace(string(buf))
        return text, fmt.Sprintf("plain ext=%s len=%d", ext, len(text)), nil
    }

    // Unknown/binary — return an error to signal unsupported type
    return nil, "", errors.New("unsupported file type; provide PDF/HTML/text/CSV/JSON/YAML")
}

func prependLog(kind, logs string) string {
    if logs == "" { return kind }
    return kind + " " + logs
}
