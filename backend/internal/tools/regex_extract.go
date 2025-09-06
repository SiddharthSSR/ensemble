package tools

import (
    "context"
    "fmt"
    "regexp"
    "strings"
)

// RegexExtractTool finds all matches of a regex pattern in text.
// Inputs:
// - text: string (required)
// - pattern: string (required; supports named groups ?P<name>)
// - flags: string (optional; combine i,m,s)
// - limit: number (optional; max matches, default 100)
// Output: if named groups -> []object; else -> [][]string (full + submatches)
type RegexExtractTool struct{}

func (t *RegexExtractTool) Name() string { return "regex_extract" }

func (t *RegexExtractTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    text, _ := inputs["text"].(string)
    pat, _ := inputs["pattern"].(string)
    if strings.TrimSpace(text) == "" { return []any{}, "", nil }
    if strings.TrimSpace(pat) == "" { return nil, "", fmt.Errorf("missing pattern") }

    flags, _ := inputs["flags"].(string)
    prefix := ""
    if flags != "" {
        var f []string
        flags = strings.ToLower(flags)
        if strings.Contains(flags, "i") { f = append(f, "i") }
        if strings.Contains(flags, "m") { f = append(f, "m") }
        if strings.Contains(flags, "s") { f = append(f, "s") }
        if len(f) > 0 { prefix = "(?" + strings.Join(f, "") + ")" }
    }
    rx, err := regexp.Compile(prefix + pat)
    if err != nil { return nil, "", err }

    limit := 100
    if v, ok := inputs["limit"].(float64); ok && v > 0 { limit = int(v) }

    // detect named groups
    names := rx.SubexpNames()
    hasNamed := false
    for _, n := range names { if n != "" { hasNamed = true; break } }

    var out any
    if hasNamed {
        var rows []map[string]string
        it := rx.FindAllStringSubmatchIndex(text, limit)
        for _, idx := range it {
            row := map[string]string{}
            for gi := 1; gi < len(names); gi++ { // 0 is the full match
                name := names[gi]
                if name == "" { continue }
                s := idx[2*gi]
                e := idx[2*gi+1]
                if s >= 0 && e >= 0 && s <= e && e <= len(text) {
                    row[name] = text[s:e]
                }
            }
            rows = append(rows, row)
        }
        out = rows
    } else {
        var rows [][]string
        m := rx.FindAllStringSubmatch(text, limit)
        if m != nil { rows = m }
        out = rows
    }
    return out, fmt.Sprintf("matches<=%d", limit), nil
}

