package orchestrator

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "sync"
    "time"

    "github.com/example/agent-orchestrator/internal/agents"
    "github.com/example/agent-orchestrator/internal/models"
    "github.com/example/agent-orchestrator/internal/tools"
    "regexp"
    "os"
    "strconv"
)

type Orchestrator struct {
    Planner  agents.Planner
    Executor agents.Executor
    Verifier agents.Verifier

    tasksMu sync.RWMutex
    tasks   map[string]*models.Task

    hub *Hub
}

func New(planner agents.Planner, executor agents.Executor, verifier agents.Verifier) *Orchestrator {
    return &Orchestrator{
        Planner:  planner,
        Executor: executor,
        Verifier: verifier,
        tasks:    map[string]*models.Task{},
        hub:      NewHub(),
    }
}

func (o *Orchestrator) CreateTask(id string, query string, contextMap map[string]any) *models.Task {
    t := &models.Task{ID: id, Query: query, Context: contextMap, Status: models.StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
    o.tasksMu.Lock()
    o.tasks[id] = t
    o.tasksMu.Unlock()
    o.hub.Publish(id, Event{Event: "task_status", TaskID: id, Payload: map[string]any{"status": t.Status}})
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
    o.hub.Publish(id, Event{Event: "task_status", TaskID: id, Payload: map[string]any{"status": t.Status}})

    // Plan
    plan, err := o.Planner.Plan(ctx, t)
    if err != nil {
        t.Status = models.StatusFailed
        t.UpdatedAt = time.Now()
        o.hub.Publish(id, Event{Event: "task_status", TaskID: id, Payload: map[string]any{"status": t.Status, "error": err.Error()}})
        return err
    }
    t.Plan = plan
    o.hub.Publish(id, Event{Event: "plan", TaskID: id, Payload: plan})

    // Sequential execution MVP
    resultsByID := map[string]*models.Result{}
    for _, step := range plan.Steps {
        step.Status = models.StatusRunning
        t.UpdatedAt = time.Now()
        o.hub.Publish(id, Event{Event: "step_status", TaskID: id, Payload: step})
        // resolve input references from prior step outputs (deeply)
        step.Inputs = resolveInputsDeep(step.Inputs, resultsByID)
        // attach token streaming callback for LLM tools
        appender := o.hub.TokenAppender(id)
        subCtx := context.WithValue(ctx, tools.CtxTokenCallbackKey, tools.TokenCallback(func(chunk string) {
            appender(step.ID, chunk)
        }))
        res, _ := o.Executor.Execute(subCtx, step)
        verified, _ := o.Verifier.Verify(ctx, t, step, res)
        res.Verified = verified
        if !verified || res.Error != "" {
            step.Status = models.StatusFailed
            if t.Results == nil { t.Results = []*models.Result{} }
            t.Results = append(t.Results, res)
            t.Status = models.StatusFailed
            t.UpdatedAt = time.Now()
            o.hub.Publish(id, Event{Event: "result", TaskID: id, Payload: o.previewResult(res)})
            o.hub.Publish(id, Event{Event: "step_status", TaskID: id, Payload: step})
            o.hub.Publish(id, Event{Event: "task_status", TaskID: id, Payload: map[string]any{"status": t.Status}})
            return nil
        }
        res.Verified = true
        resultsByID[step.ID] = res
        if t.Results == nil { t.Results = []*models.Result{} }
        t.Results = append(t.Results, res)
        step.Status = models.StatusSuccess
        t.UpdatedAt = time.Now()
        o.hub.Publish(id, Event{Event: "result", TaskID: id, Payload: o.previewResult(res)})
        o.hub.Publish(id, Event{Event: "step_status", TaskID: id, Payload: step})
    }

    // flush token coalescer
    o.hub.StopTokenAppender(id)
    t.Status = models.StatusSuccess
    t.UpdatedAt = time.Now()
    o.hub.Publish(id, Event{Event: "task_status", TaskID: id, Payload: map[string]any{"status": t.Status}})
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
    resultsByID := map[string]*models.Result{}
    for _, step := range t.Plan.Steps {
        step.Status = models.StatusRunning
        t.UpdatedAt = time.Now()
        o.hub.Publish(id, Event{Event: "step_status", TaskID: id, Payload: step})
        step.Inputs = resolveInputsDeep(step.Inputs, resultsByID)
        appender := o.hub.TokenAppender(id)
        subCtx := context.WithValue(ctx, tools.CtxTokenCallbackKey, tools.TokenCallback(func(chunk string) {
            appender(step.ID, chunk)
        }))
        res, _ := o.Executor.Execute(subCtx, step)
        verified, _ := o.Verifier.Verify(ctx, t, step, res)
        res.Verified = verified
        if !verified || res.Error != "" {
            step.Status = models.StatusFailed
            if t.Results == nil { t.Results = []*models.Result{} }
            t.Results = append(t.Results, res)
            t.Status = models.StatusFailed
            t.UpdatedAt = time.Now()
            o.hub.Publish(id, Event{Event: "result", TaskID: id, Payload: o.previewResult(res)})
            o.hub.Publish(id, Event{Event: "step_status", TaskID: id, Payload: step})
            o.hub.Publish(id, Event{Event: "task_status", TaskID: id, Payload: map[string]any{"status": t.Status}})
            return nil
        }
        res.Verified = true
        resultsByID[step.ID] = res
        if t.Results == nil { t.Results = []*models.Result{} }
        t.Results = append(t.Results, res)
        step.Status = models.StatusSuccess
        t.UpdatedAt = time.Now()
        o.hub.Publish(id, Event{Event: "result", TaskID: id, Payload: o.previewResult(res)})
        o.hub.Publish(id, Event{Event: "step_status", TaskID: id, Payload: step})
    }
    o.hub.StopTokenAppender(id)
    t.Status = models.StatusSuccess
    t.UpdatedAt = time.Now()
    o.hub.Publish(id, Event{Event: "task_status", TaskID: id, Payload: map[string]any{"status": t.Status}})
    return nil
}

// Subscribe returns a channel carrying JSON-encoded Event payloads for a specific task.
// The caller must call the returned unsubscribe func when done.
func (o *Orchestrator) Subscribe(taskID string) (<-chan []byte, func()) {
    ch, unsub := o.hub.Subscribe(taskID)
    return ch, unsub
}

// resolveInputs replaces any string value exactly matching {{step:ID.output}} with the
// stringified output of that prior step, if available.
func resolveInputsDeep(inputs map[string]any, resultsByID map[string]*models.Result) map[string]any {
    if inputs == nil { return nil }
    out := make(map[string]any, len(inputs))
    for k, v := range inputs { out[k] = resolveValueDeep(v, resultsByID) }
    return out
}

func resolveValueDeep(v any, resultsByID map[string]*models.Result) any {
    switch t := v.(type) {
    case string:
        return replacePlaceholders(t, resultsByID)
    case map[string]any:
        m := make(map[string]any, len(t))
        for k2, v2 := range t { m[k2] = resolveValueDeep(v2, resultsByID) }
        return m
    case []any:
        a := make([]any, len(t))
        for i := range t { a[i] = resolveValueDeep(t[i], resultsByID) }
        return a
    default:
        return v
    }
}

func replacePlaceholders(s string, resultsByID map[string]*models.Result) string {
    re := regexp.MustCompile(`\{\{step:([a-zA-Z0-9_\-]+)\.output\}\}`)
    return re.ReplaceAllStringFunc(s, func(m string) string {
        match := re.FindStringSubmatch(m)
        if len(match) != 2 { return m }
        id := match[1]
        if res, ok := resultsByID[id]; ok && res != nil {
            return stringifyOutput(res.Output)
        }
        return fmt.Sprintf("(missing output from %s)", id)
    })
}

func stringifyOutput(v any) string {
    switch t := v.(type) {
    case string:
        return t
    default:
        b, _ := json.Marshal(t)
        return string(b)
    }
}

func (o *Orchestrator) previewResult(res *models.Result) map[string]any {
    max := 20000
    if v := os.Getenv("PREVIEW_MAX_BYTES"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { max = n } }
    preview := res.Output
    var size int
    var truncated bool
    switch t := res.Output.(type) {
    case string:
        size = len(t)
        if size > max { preview = t[:max]; truncated = true }
    default:
        s := stringifyOutput(res.Output)
        size = len(s)
        if size > max { s = s[:max]; truncated = true }
        preview = s
    }
    out := map[string]any{
        "step_id":  res.StepID,
        "output":   preview,
        "logs":     res.Logs,
        "verified": res.Verified,
        "error":    res.Error,
    }
    // annotate preview metadata
    if truncated { out["preview_truncated"] = true }
    out["bytes_total"] = size
    return out
}
