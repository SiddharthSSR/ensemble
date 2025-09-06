package tools

import (
    "context"
    "encoding/base64"
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
    truncated := false
    if len(buf) > max {
        buf = buf[:max]
        truncated = true
    }

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
    isCSV := ext == "csv" || strings.Contains(strings.ToLower(ctype), "csv") || strings.Contains(strings.ToLower(ctype), "ms-excel")
    if ext == "txt" || ext == "md" || ext == "markdown" || isCSV || ext == "json" || ext == "log" || ext == "yaml" || ext == "yml" ||
        strings.Contains(strings.ToLower(ctype), "text/") || strings.Contains(strings.ToLower(ctype), "json") || strings.Contains(strings.ToLower(ctype), "yaml") || looksLikeText(buf) {
        // Return as string (no parsing); callers can choose summarize or csv_parse/regex_extract next
        text := strings.TrimSpace(string(buf))
        meta := []string{}
        if ext != "" { meta = append(meta, "ext="+ext) }
        if ctype != "" { meta = append(meta, "ctype="+ctype) }
        meta = append(meta, fmt.Sprintf("len=%d", len(text)))
        if truncated { meta = append(meta, "truncated=true") }
        return text, strings.Join(meta, " "), nil
    }

    // Unknown/binary — still return raw text safely trimmed to avoid failing flows
    text := strings.TrimSpace(string(buf))
    log := fmt.Sprintf("binary? len=%d", len(text))
    if truncated { log += " truncated=true" }
    return text, log, nil
}

func prependLog(kind, logs string) string {
    if logs == "" { return kind }
    return kind + " " + logs
}

// looksLikeText returns true if the buffer appears to be textual data.
func looksLikeText(b []byte) bool {
    if len(b) == 0 { return false }
    n := len(b)
    // sample up to first 4096 bytes
    if n > 4096 { n = 4096 }
    var printable, zeros int
    for i := 0; i < n; i++ {
        c := b[i]
        if c == 0 { zeros++; continue }
        // allow common whitespace and printable ASCII
        if (c >= 32 && c <= 126) || c == '\n' || c == '\r' || c == '\t' { printable++ }
    }
    if zeros > 0 { return false }
    // at least 80% printable suggests text
    return printable*100/n >= 80
}
