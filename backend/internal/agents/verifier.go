package agents

import (
    "context"
    "strings"

    "github.com/example/agent-orchestrator/internal/models"
)

type Verifier interface {
    Verify(ctx context.Context, task *models.Task, step *models.Step, res *models.Result) (bool, string)
}

// SimpleVerifier performs basic checks; placeholder for LLM-backed verifier.
type SimpleVerifier struct{}

func (v *SimpleVerifier) Verify(ctx context.Context, task *models.Task, step *models.Step, res *models.Result) (bool, string) {
    if res.Error != "" {
        return false, "execution error returned"
    }
    // Naive: non-empty output passes. If echo tool, ensure it contains text.
    if step.Tool == "echo" {
        text, _ := step.Inputs["text"].(string)
        outStr := toString(res.Output)
        if strings.Contains(outStr, text) && outStr != "" {
            return true, "ok"
        }
        return false, "echo output mismatch"
    }
    return res.Output != nil, "ok"
}

func toString(v any) string {
    switch t := v.(type) {
    case string:
        return t
    default:
        return ""
    }
}

