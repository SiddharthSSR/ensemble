package agents

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/example/agent-orchestrator/internal/models"
    gem "github.com/example/agent-orchestrator/internal/providers/gemini"
)

// LLMVerifier asks an LLM to judge if a result meets the step intent.
type LLMVerifier struct { Client gem.Client }

func (v *LLMVerifier) Verify(ctx context.Context, task *models.Task, step *models.Step, res *models.Result) (bool, string) {
    if res.Error != "" { return false, "execution error" }
    outStr := stringify(res.Output)
    ok, reason, err := v.Client.Verify(ctx, buildVerifyPrompt(task, step), outStr)
    if err != nil { return false, err.Error() }
    return ok, reason
}

func buildVerifyPrompt(task *models.Task, step *models.Step) string {
    b, _ := json.Marshal(map[string]any{"query": task.Query, "context": task.Context, "step": step})
    return fmt.Sprintf(`You are a strict verifier. Given the task and step, return whether the output satisfies the step's intent and is relevant.
Respond with JSON: {"ok": true|false, "reason": "..."}.
Task and step: %s`, string(b))
}

func stringify(v any) string {
    switch t := v.(type) {
    case string:
        return t
    default:
        b, _ := json.Marshal(t)
        return string(b)
    }
}

