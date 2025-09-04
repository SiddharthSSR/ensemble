package tools

import (
    "context"
    "fmt"
)

type EchoTool struct{}

func (e *EchoTool) Name() string { return "echo" }

func (e *EchoTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    text, _ := inputs["text"].(string)
    out := fmt.Sprintf("echo: %s", text)
    return out, "", nil
}

