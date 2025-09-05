package tools

import (
    "context"
    "fmt"

    "github.com/example/agent-orchestrator/internal/providers/llm"
)

type SummarizeTool struct{ Client llm.Client }

func (s *SummarizeTool) Name() string { return "summarize" }

func (s *SummarizeTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    text, _ := inputs["text"].(string)
    if text == "" {
        return nil, "", fmt.Errorf("missing text")
    }
    prompt := fmt.Sprintf("Summarize the following text in a concise way (3-5 bullet points or a short paragraph). Focus on key facts.\n\nText:\n%s", text)
    if cb, ok := ctx.Value(CtxTokenCallbackKey).(TokenCallback); ok && cb != nil {
        var acc string
        err := s.Client.GenerateTextStream(ctx, prompt, func(chunk string) error { acc += chunk; cb(chunk); return nil })
        if err != nil { return nil, "", err }
        return acc, "", nil
    }
    out, err := s.Client.GenerateText(ctx, prompt)
    if err != nil { return nil, "", err }
    return out, "", nil
}
