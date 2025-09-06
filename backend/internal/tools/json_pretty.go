package tools

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
)

// JSONPrettyTool validates and pretty-prints a JSON string.
// Inputs:
// - json: string (required) — raw JSON text
// Output: string (indented JSON)
type JSONPrettyTool struct{}

func (t *JSONPrettyTool) Name() string { return "json_pretty" }

func (t *JSONPrettyTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    raw, _ := inputs["json"].(string)
    if strings.TrimSpace(raw) == "" { return "", "", fmt.Errorf("missing json") }
    var v any
    if err := json.Unmarshal([]byte(raw), &v); err != nil { return nil, "", fmt.Errorf("invalid json: %w", err) }
    out, err := json.MarshalIndent(v, "", "  ")
    if err != nil { return nil, "", err }
    return string(out), "", nil
}

