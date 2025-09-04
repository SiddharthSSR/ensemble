package agents

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/example/agent-orchestrator/internal/models"
    "github.com/example/agent-orchestrator/internal/providers/llm"
)

// LLMPlanner uses an LLM provider to produce a structured plan.
type LLMPlanner struct { Client llm.Client }

type llmStep struct {
    ID          string                 `json:"id"`
    Description string                 `json:"description"`
    Tool        string                 `json:"tool"`
    Inputs      map[string]any         `json:"inputs,omitempty"`
    Deps        []string               `json:"deps,omitempty"`
}

func (p *LLMPlanner) Plan(ctx context.Context, task *models.Task) (*models.Plan, error) {
    prompt := buildPlanPrompt(task)
    raw, err := p.Client.GeneratePlan(ctx, prompt)
    if err != nil || strings.TrimSpace(raw) == "" {
        // Fallback to trivial plan
        return trivialPlan(task), nil
    }
    var steps []llmStep
    text := normalizeJSONText(raw)
    if err := json.Unmarshal([]byte(text), &steps); err != nil {
        // Be lenient; sometimes models wrap JSON. Try to extract first [] block.
        if arr := extractJSONArray(text); arr != "" {
            if err2 := json.Unmarshal([]byte(arr), &steps); err2 != nil {
                return trivialPlan(task), nil
            }
        } else {
            return trivialPlan(task), nil
        }
    }
    if len(steps) == 0 {
        return trivialPlan(task), nil
    }
    out := make([]*models.Step, 0, len(steps))
    for i, s := range steps {
        id := s.ID
        if id == "" {
            id = fmt.Sprintf("step%d", i+1)
        }
        out = append(out, &models.Step{
            ID:          id,
            Description: s.Description,
            Tool:        s.Tool,
            Inputs:      s.Inputs,
            Deps:        s.Deps,
            Status:      models.StatusPending,
        })
    }
    return &models.Plan{Steps: out}, nil
}

func buildPlanPrompt(task *models.Task) string {
    return fmt.Sprintf(`You are a planning agent for a constrained tool runner.
Output ONLY a JSON array of step objects, no prose, no code fences.

Tools (you MUST stick to these):
- echo: inputs {"text": string}
- http_get: inputs {"url": string}
- summarize: inputs {"text": string}

Rules:
- Produce 1–3 ordered steps. Prefer 2 steps when helpful.
- Use "deps" to express order (e.g., step2 depends on step1).
- To pass the output of a previous step to a later step, set a string input to the exact template: {{step:ID.output}}
- If the query contains or implies a URL, first add an http_get step using that URL, then add a summarize step with {"text": "{{step:step1.output}}"} (adjust ID as needed) to produce a concise summary.
- If there is no URL, use 1–2 echo steps: first restate or clarify the intent; optionally add a second echo suggesting a next action.

Schema for each step: {"id": "stepN", "description": "...", "tool": "echo"|"http_get"|"summarize", "inputs": { ... }, "deps": ["stepK"]}

User query: %s
Context: %v`, task.Query, task.Context)
}

func trivialPlan(task *models.Task) *models.Plan {
    lower := strings.ToLower(task.Query)
    if strings.Contains(lower, "http") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
        return &models.Plan{Steps: []*models.Step{{
            ID:          "step1",
            Description: "HTTP GET a URL",
            Tool:        "http_get",
            Inputs:      map[string]any{"url": task.Query},
            Status:      models.StatusPending,
        }}}
    }
    return &models.Plan{Steps: []*models.Step{{
        ID:          "step1",
        Description: "Echo the query",
        Tool:        "echo",
        Inputs:      map[string]any{"text": task.Query},
        Status:      models.StatusPending,
    }}}
}

func extractJSONArray(s string) string {
    // crude extractor for the first top-level JSON array in a string
    start := strings.Index(s, "[")
    if start == -1 { return "" }
    depth := 0
    for i := start; i < len(s); i++ {
        if s[i] == '[' { depth++ }
        if s[i] == ']' {
            depth--
            if depth == 0 {
                return s[start:i+1]
            }
        }
    }
    return ""
}

func normalizeJSONText(s string) string {
    t := strings.TrimSpace(s)
    // Strip code fences like ```json ... ```
    if strings.HasPrefix(t, "```") {
        // remove first fence
        t = strings.TrimPrefix(t, "```")
        // drop possible language hint, e.g., json
        if idx := strings.IndexByte(t, '\n'); idx != -1 {
            t = t[idx+1:]
        }
        // remove trailing fence if present
        if j := strings.LastIndex(t, "```"); j != -1 {
            t = t[:j]
        }
        t = strings.TrimSpace(t)
    }
    // If it's not starting with '[' try to extract the first JSON array
    if !strings.HasPrefix(strings.TrimSpace(t), "[") {
        if arr := extractJSONArray(t); arr != "" {
            return arr
        }
    }
    return t
}
