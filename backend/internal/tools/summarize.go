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
    out, err := s.Client.GenerateText(ctx, prompt)
    if err != nil { return nil, "", err }
    return out, "", nil
}

