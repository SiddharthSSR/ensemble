package tools

import (
    "context"
    "fmt"

    "github.com/example/agent-orchestrator/internal/providers/llm"
)

type LLMAnswerTool struct{ Client llm.Client }

func (t *LLMAnswerTool) Name() string { return "llm_answer" }

func (t *LLMAnswerTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    // accept either "text" or "question"
    q, _ := inputs["text"].(string)
    if q == "" { q, _ = inputs["question"].(string) }
    if q == "" { return nil, "", fmt.Errorf("missing text/question") }
    // optional instructions
    inst, _ := inputs["instructions"].(string)
    prompt := q
    if inst != "" { prompt = inst + "\n\nQuestion:\n" + q }
    if cb, ok := ctx.Value(CtxTokenCallbackKey).(TokenCallback); ok && cb != nil {
        var acc string
        err := t.Client.GenerateTextStream(ctx, prompt, func(chunk string) error { acc += chunk; cb(chunk); return nil })
        if err != nil { return nil, "", err }
        return acc, "", nil
    }
    ans, err := t.Client.GenerateText(ctx, prompt)
    if err != nil { return nil, "", err }
    return ans, "", nil
}
