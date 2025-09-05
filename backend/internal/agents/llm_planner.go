package agents

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "log"
    "os"

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
        if os.Getenv("LLM_DEBUG") == "1" && err != nil {
            log.Printf("LLMPlanner: generate error: %v", err)
        }
        // Fallback to trivial plan
        return trivialPlan(task), nil
    }
    var steps []llmStep
    text := normalizeJSONText(raw)
    if err := json.Unmarshal([]byte(text), &steps); err != nil {
        if os.Getenv("LLM_DEBUG") == "1" {
            log.Printf("LLMPlanner: json unmarshal failed, raw=%.200q err=%v", text, err)
        }
        // Try extracting first [] array
        if arr := extractJSONArray(text); arr != "" {
            if err2 := json.Unmarshal([]byte(arr), &steps); err2 != nil {
                if os.Getenv("LLM_DEBUG") == "1" { log.Printf("LLMPlanner: array extraction also failed: %v", err2) }
            }
        }
        // Try wrapper object {"steps": [...]}
        if len(steps) == 0 {
            var wrapper struct{ Steps []llmStep `json:"steps"` }
            if err3 := json.Unmarshal([]byte(text), &wrapper); err3 == nil && len(wrapper.Steps) > 0 {
                steps = wrapper.Steps
            }
        }
        if len(steps) == 0 { return trivialPlan(task), nil }
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
- html_to_text: inputs {"html": string}
- summarize: inputs {"text": string}
- llm_answer: inputs {"text": string}
 - http_post_json: inputs {"url": string, "json": any}

Rules:
- Produce 1–3 ordered steps. Prefer 2 steps when helpful.
- Use "deps" to express order (e.g., step2 depends on step1).
- To pass the output of a previous step to a later step, set a string input to the exact template: {{step:ID.output}}
- If the query contains or implies a URL, plan: (1) http_get(url) -> (2) html_to_text(html="{{step:step1.output}}") -> (3) summarize(text="{{step:step2.output}}").
- If the query starts with "summarize:" or "summarise:", use a single summarize step with {"text": "<rest of query>"}.
- If the query suggests calling a JSON API (mentions POST/JSON/payload) and includes a URL and a simple JSON object, use a single http_post_json step with that URL and JSON.
- If there is no URL and it is a direct question, use a single llm_answer step with {"text": "<the query>"}.

Schema for each step: {"id": "stepN", "description": "...", "tool": "echo"|"http_get"|"html_to_text"|"summarize"|"llm_answer"|"http_post_json", "inputs": { ... }, "deps": ["stepK"]}

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
