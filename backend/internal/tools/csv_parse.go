package tools

import (
    "context"
    "encoding/csv"
    "errors"
    "fmt"
    "io"
    "strings"
)

// CSVParseTool converts CSV text into structured JSON.
// Inputs:
// - csv: string (required)
// - delimiter: string, one rune (optional; default ',')
// - headers: []string (optional; overrides first-row headers)
// - has_header: bool (optional; default true if no headers provided)
// Output: []object (row objects with string values)
type CSVParseTool struct{}

func (t *CSVParseTool) Name() string { return "csv_parse" }

func (t *CSVParseTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    raw, _ := inputs["csv"].(string)
    if strings.TrimSpace(raw) == "" { return []map[string]string{}, "", nil }
    rdr := csv.NewReader(strings.NewReader(raw))
    // allow ragged rows
    rdr.FieldsPerRecord = -1

    // delimiter
    if d, ok := inputs["delimiter"].(string); ok && d != "" {
        r := []rune(d)
        if len(r) != 1 { return nil, "", fmt.Errorf("delimiter must be a single character") }
        rdr.Comma = r[0]
    }

    // headers input
    var headers []string
    if hv, ok := inputs["headers"].([]any); ok {
        for _, v := range hv { if s, ok := v.(string); ok { headers = append(headers, s) } }
    }
    hasHeader := true
    if _, ok := inputs["has_header"]; ok {
        if b, ok := inputs["has_header"].(bool); ok { hasHeader = b }
        if f, ok := inputs["has_header"].(float64); ok { hasHeader = f != 0 }
    }

    var err error
    if len(headers) == 0 && hasHeader {
        headers, err = rdr.Read()
        if err != nil { return nil, "", err }
        for i := range headers { headers[i] = strings.TrimSpace(headers[i]) }
    }

    // read all rows (ensure non-nil slice so JSON is [] not null)
    out := make([]map[string]string, 0, 64)
    idx := 0
    for {
        rec, err := rdr.Read()
        if err != nil {
            if errors.Is(err, io.EOF) { break }
            // Some CSVs may end with a partial line; treat as EOF
            if strings.Contains(err.Error(), "EOF") { break }
            return out, "", err
        }
        // derive headers if not provided
        if len(headers) == 0 {
            headers = make([]string, len(rec))
            for i := range rec { headers[i] = fmt.Sprintf("c%d", i+1) }
        }
        row := map[string]string{}
        for i := range headers {
            var v string
            if i < len(rec) { v = rec[i] }
            row[headers[i]] = v
        }
        out = append(out, row)
        idx++
    }
    return out, fmt.Sprintf("rows=%d cols=%d", len(out), len(headers)), nil
}
