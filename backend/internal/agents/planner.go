package agents

import (
    "context"
    "strings"

    "github.com/example/agent-orchestrator/internal/models"
)

type Planner interface {
    Plan(ctx context.Context, task *models.Task) (*models.Plan, error)
}

// MockPlanner is a simple rule-based planner for MVP.
type MockPlanner struct{}

func (m *MockPlanner) Plan(ctx context.Context, task *models.Task) (*models.Plan, error) {
    q := strings.ToLower(task.Query)
    // Prefer richer defaults: URL -> http_get -> html_to_text -> summarize, else llm_answer
    if strings.Contains(q, "http") || strings.HasPrefix(q, "http://") || strings.HasPrefix(q, "https://") {
        return &models.Plan{Steps: []*models.Step{
            {
                ID:          "step1",
                Description: "HTTP GET a URL",
                Tool:        "http_get",
                Inputs:      map[string]any{"url": task.Query},
                Status:      models.StatusPending,
            },
            {
                ID:          "step2",
                Description: "Convert HTML to text",
                Tool:        "html_to_text",
                Inputs:      map[string]any{"html": "{{step:step1.output}}"},
                Deps:        []string{"step1"},
                Status:      models.StatusPending,
            },
            {
                ID:          "step3",
                Description: "Summarize content",
                Tool:        "summarize",
                Inputs:      map[string]any{"text": "{{step:step2.output}}"},
                Deps:        []string{"step2"},
                Status:      models.StatusPending,
            },
        }}, nil
    }
    return &models.Plan{Steps: []*models.Step{{
        ID:          "step1",
        Description: "Answer with LLM",
        Tool:        "llm_answer",
        Inputs:      map[string]any{"text": task.Query},
        Status:      models.StatusPending,
    }}}, nil
}
