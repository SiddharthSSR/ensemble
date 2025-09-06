package tools

import (
    "context"
    "errors"
    "fmt"
)

// CallTool delegates execution to another registered tool using provided inputs.
// Inputs:
// - tool: string (required) — name of a registered tool
// - inputs: object (optional) — inputs map passed to the delegate tool
// Safety: recursion into call_tool is blocked.
type CallTool struct { Registry *Registry }

func (t *CallTool) Name() string { return "call_tool" }

func (t *CallTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    if t.Registry == nil { return nil, "", errors.New("registry not set") }
    name, _ := inputs["tool"].(string)
    if name == "" { return nil, "", fmt.Errorf("missing tool name") }
    if name == t.Name() { return nil, "", fmt.Errorf("recursive call to %q is not allowed", t.Name()) }
    delegate, ok := t.Registry.Get(name)
    if !ok { return nil, "", fmt.Errorf("unknown tool: %s", name) }

    var child map[string]any
    if m, ok := inputs["inputs"].(map[string]any); ok { child = m } else { child = map[string]any{} }
    out, logs, err := delegate.Execute(ctx, child)
    if logs != "" { logs = fmt.Sprintf("delegated=%s %s", name, logs) } else { logs = fmt.Sprintf("delegated=%s", name) }
    return out, logs, err
}

