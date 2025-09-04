package orchestrator

import (
    "context"
    "errors"
    "sync"
    "time"

    "github.com/example/agent-orchestrator/internal/agents"
    "github.com/example/agent-orchestrator/internal/models"
)

type Orchestrator struct {
    Planner  agents.Planner
    Executor agents.Executor
    Verifier agents.Verifier

    tasksMu sync.RWMutex
    tasks   map[string]*models.Task
}

func New(planner agents.Planner, executor agents.Executor, verifier agents.Verifier) *Orchestrator {
    return &Orchestrator{
        Planner:  planner,
        Executor: executor,
        Verifier: verifier,
        tasks:    map[string]*models.Task{},
    }
}

func (o *Orchestrator) CreateTask(id string, query string, contextMap map[string]any) *models.Task {
    t := &models.Task{ID: id, Query: query, Context: contextMap, Status: models.StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
    o.tasksMu.Lock()
    o.tasks[id] = t
    o.tasksMu.Unlock()
    return t
}

func (o *Orchestrator) GetTask(id string) (*models.Task, bool) {
    o.tasksMu.RLock()
    t, ok := o.tasks[id]
    o.tasksMu.RUnlock()
    return t, ok
}

func (o *Orchestrator) ListTasks() []*models.Task {
    o.tasksMu.RLock()
    out := make([]*models.Task, 0, len(o.tasks))
    for _, t := range o.tasks {
        out = append(out, t)
    }
    o.tasksMu.RUnlock()
    return out
}

func (o *Orchestrator) Start(ctx context.Context, id string) error {
    t, ok := o.GetTask(id)
    if !ok {
        return errors.New("task not found")
    }
    t.Status = models.StatusRunning
    t.UpdatedAt = time.Now()

    // Plan
    plan, err := o.Planner.Plan(ctx, t)
    if err != nil {
        t.Status = models.StatusFailed
        t.UpdatedAt = time.Now()
        return err
    }
    t.Plan = plan

    // Sequential execution MVP
    for _, step := range plan.Steps {
        step.Status = models.StatusRunning
        t.UpdatedAt = time.Now()
        res, _ := o.Executor.Execute(ctx, step)
        verified, _ := o.Verifier.Verify(ctx, t, step, res)
        res.Verified = verified
        if !verified || res.Error != "" {
            step.Status = models.StatusFailed
            if t.Results == nil { t.Results = []*models.Result{} }
            t.Results = append(t.Results, res)
            t.Status = models.StatusFailed
            t.UpdatedAt = time.Now()
            return nil
        }
        res.Verified = true
        if t.Results == nil { t.Results = []*models.Result{} }
        t.Results = append(t.Results, res)
        step.Status = models.StatusSuccess
        t.UpdatedAt = time.Now()
    }

    t.Status = models.StatusSuccess
    t.UpdatedAt = time.Now()
    return nil
}

// PlanOnly computes a plan for a task and stores it without executing.
func (o *Orchestrator) PlanOnly(ctx context.Context, id string) (*models.Plan, error) {
    t, ok := o.GetTask(id)
    if !ok {
        return nil, errors.New("task not found")
    }
    plan, err := o.Planner.Plan(ctx, t)
    if err != nil {
        t.Status = models.StatusFailed
        t.UpdatedAt = time.Now()
        return nil, err
    }
    t.Plan = plan
    t.Status = models.StatusPlanned
    t.UpdatedAt = time.Now()
    return plan, nil
}

// ExecutePlan executes an already-generated plan on the task without re-planning.
func (o *Orchestrator) ExecutePlan(ctx context.Context, id string) error {
    t, ok := o.GetTask(id)
    if !ok {
        return errors.New("task not found")
    }
    if t.Plan == nil || len(t.Plan.Steps) == 0 {
        return errors.New("no plan to execute")
    }
    t.Status = models.StatusRunning
    t.UpdatedAt = time.Now()
    // Sequential execution MVP
    for _, step := range t.Plan.Steps {
        step.Status = models.StatusRunning
        t.UpdatedAt = time.Now()
        res, _ := o.Executor.Execute(ctx, step)
        verified, _ := o.Verifier.Verify(ctx, t, step, res)
        res.Verified = verified
        if !verified || res.Error != "" {
            step.Status = models.StatusFailed
            if t.Results == nil { t.Results = []*models.Result{} }
            t.Results = append(t.Results, res)
            t.Status = models.StatusFailed
            t.UpdatedAt = time.Now()
            return nil
        }
        res.Verified = true
        if t.Results == nil { t.Results = []*models.Result{} }
        t.Results = append(t.Results, res)
        step.Status = models.StatusSuccess
        t.UpdatedAt = time.Now()
    }
    t.Status = models.StatusSuccess
    t.UpdatedAt = time.Now()
    return nil
}
