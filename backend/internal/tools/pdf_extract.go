package tools

import (
    "context"
    "encoding/base64"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
    pdfx "github.com/ledongthuc/pdf"
)

type PDFExtractTool struct{}

func (t *PDFExtractTool) Name() string { return "pdf_extract" }

func (t *PDFExtractTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    dataB64, _ := inputs["data_base64"].(string)
    if dataB64 == "" {
        return nil, "", fmt.Errorf("missing data_base64")
    }
    maxBytes := getInt(inputs, "max_bytes", envInt("PDF_MAX_BYTES", 20*1024*1024))
    maxPages := getInt(inputs, "max_pages", envInt("PDF_MAX_PAGES", 20))
    deadline := time.Now().Add(time.Duration(envInt("PDF_TIMEOUT_MS", 60000)) * time.Millisecond)
    // decode base64 (allow data: URIs)
    if i := strings.Index(dataB64, ","); i != -1 { dataB64 = dataB64[i+1:] }
    buf, err := base64.StdEncoding.DecodeString(dataB64)
    if err != nil { return nil, "", fmt.Errorf("invalid base64: %w", err) }
    if len(buf) > maxBytes { return nil, "", fmt.Errorf("pdf too large: %d bytes > limit %d", len(buf), maxBytes) }
    // write to temp file because pdf lib expects a path
    dir := os.TempDir()
    path := filepath.Join(dir, fmt.Sprintf("pdf_%d.pdf", os.Getpid()))
    if err := os.WriteFile(path, buf, 0600); err != nil { return nil, "", err }
    defer os.Remove(path)
    // open and extract text
    f, r, err := pdfx.Open(path)
    if err != nil { return nil, "", err }
    defer f.Close()
    totalPages := r.NumPage()
    // pages range e.g. "1-3,7"
    pagesSpec, _ := inputs["pages"].(string)
    selected := expandPages(pagesSpec, totalPages)
    if len(selected) == 0 { for i:=1;i<=totalPages;i++ { selected = append(selected, i) } }
    if len(selected) > maxPages { selected = selected[:maxPages] }

    var out strings.Builder
    // streaming callback support
    var cb TokenCallback
    if v := ctx.Value(CtxTokenCallbackKey); v != nil {
        if fn, ok := v.(TokenCallback); ok { cb = fn }
    }
    for _, page := range selected {
        if time.Now().After(deadline) { return nil, "", errors.New("pdf extraction timeout") }
        p := r.Page(page)
        txt, _ := p.GetPlainText(nil)
        t := strings.TrimSpace(txt)
        if t != "" {
            if cb != nil { cb(fmt.Sprintf("\n\n--- Page %d ---\n%s", page, t)) }
            out.WriteString(t)
            out.WriteString("\n\n")
        }
    }
    text := strings.TrimSpace(out.String())
    return text, fmt.Sprintf("pages=%d/%d bytes=%d", len(selected), totalPages, len(buf)), nil
}

func envInt(key string, def int) int {
    if v := os.Getenv(key); v != "" { if n, err := strconv.Atoi(v); err==nil { return n } }
    return def
}

func getInt(m map[string]any, key string, def int) int {
    if v, ok := m[key]; ok {
        switch t := v.(type) {
        case float64: return int(t)
        case int: return t
        case string: if n, err := strconv.Atoi(t); err==nil { return n }
        }
    }
    return def
}

func expandPages(spec string, total int) []int {
    var out []int
    spec = strings.TrimSpace(spec)
    if spec == "" { return out }
    parts := strings.Split(spec, ",")
    seen := map[int]struct{}{}
    add := func(n int) { if n>=1 && n<=total { if _,ok:=seen[n];!ok { out = append(out,n); seen[n]=struct{}{} } } }
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p == "" { continue }
        if strings.Contains(p, "-") {
            rng := strings.SplitN(p, "-", 2)
            a,_ := strconv.Atoi(strings.TrimSpace(rng[0]))
            b,_ := strconv.Atoi(strings.TrimSpace(rng[1]))
            if a>b { a,b = b,a }
            for i:=a;i<=b;i++ { add(i) }
        } else {
            n,_ := strconv.Atoi(p)
            add(n)
        }
    }
    return out
}
