package agents

import (
    "context"

    "github.com/example/agent-orchestrator/internal/models"
    "github.com/example/agent-orchestrator/internal/tools"
)

type Executor interface {
    Execute(ctx context.Context, step *models.Step) (*models.Result, error)
}

type ToolExecutor struct {
    Registry *tools.Registry
}

func (e *ToolExecutor) Execute(ctx context.Context, step *models.Step) (*models.Result, error) {
    t, ok := e.Registry.Get(step.Tool)
    if !ok {
        return &models.Result{StepID: step.ID, Error: "unknown tool: " + step.Tool}, nil
    }
    output, logs, err := t.Execute(ctx, step.Inputs)
    res := &models.Result{StepID: step.ID, Output: output, Logs: logs}
    if err != nil {
        res.Error = err.Error()
    }
    return res, nil
}

