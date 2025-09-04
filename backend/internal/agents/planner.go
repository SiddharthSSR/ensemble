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
    steps := []*models.Step{}
    if strings.Contains(q, "http") || strings.Contains(q, "fetch") || strings.Contains(q, "url") {
        steps = append(steps, &models.Step{
            ID:          "step1",
            Description: "HTTP GET a URL",
            Tool:        "http_get",
            Inputs: map[string]any{
                "url": task.Query,
            },
            Status: models.StatusPending,
        })
    } else {
        steps = append(steps, &models.Step{
            ID:          "step1",
            Description: "Echo the query",
            Tool:        "echo",
            Inputs: map[string]any{
                "text": task.Query,
            },
            Status: models.StatusPending,
        })
    }
    return &models.Plan{Steps: steps}, nil
}

