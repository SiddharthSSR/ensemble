package api

import (
    "context"
    "encoding/json"
    "log"
    "math/rand"
    "net/http"
    "time"

    "github.com/example/agent-orchestrator/internal/agents"
    "github.com/example/agent-orchestrator/internal/orchestrator"
    "github.com/example/agent-orchestrator/internal/tools"
    "github.com/example/agent-orchestrator/internal/providers/llm"
    "os"
)

var orch *orchestrator.Orchestrator

func init() {
    // Wire default components for MVP
    reg := tools.NewRegistry()
    reg.Register(&tools.EchoTool{})
    reg.Register(&tools.HTTPGetTool{})
    // LLM-backed summarize tool available when an LLM is configured (falls back to mock if not)
    reg.Register(&tools.SummarizeTool{Client: llm.NewFromEnv()})
    reg.Register(&tools.LLMAnswerTool{Client: llm.NewFromEnv()})
    reg.Register(&tools.HTMLToTextTool{})
    reg.Register(&tools.HTTPPostJSONTool{})
    // Planner selection
    var planner agents.Planner = &agents.MockPlanner{}
    if os.Getenv("USE_LLM_PLANNER") == "1" {
        planner = &agents.LLMPlanner{Client: llm.NewFromEnv()}
    }
    // Verifier selection
    var verifier agents.Verifier = &agents.SimpleVerifier{}
    if os.Getenv("USE_LLM_VERIFIER") == "1" {
        verifier = &agents.LLMVerifier{Client: llm.NewFromEnv()}
    }
    orch = orchestrator.New(planner, &agents.ToolExecutor{Registry: reg}, verifier)
}

func RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
            tasks := orch.ListTasks()
            respondJSON(w, tasks)
        case http.MethodPost:
            var req struct{
                Query string `json:"query"`
                Context map[string]any `json:"context"`
            }
            if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
            id := genID()
            t := orch.CreateTask(id, req.Query, req.Context)
            respondJSON(w, t)
        default:
            w.WriteHeader(http.StatusMethodNotAllowed)
        }
    })

    mux.HandleFunc("/tasks/start/", func(w http.ResponseWriter, r *http.Request) {
        // path: /tasks/start/{id}
        if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
        id := r.URL.Path[len("/tasks/start/"):]
        go func() {
            if err := orch.Start(context.Background(), id); err != nil {
                log.Printf("start error: %v", err)
            }
        }()
        w.WriteHeader(http.StatusAccepted)
    })

    mux.HandleFunc("/tasks/plan/", func(w http.ResponseWriter, r *http.Request) {
        // path: /tasks/plan/{id}
        if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
        id := r.URL.Path[len("/tasks/plan/"):]
        plan, err := orch.PlanOnly(r.Context(), id)
        if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
        respondJSON(w, plan)
    })

    mux.HandleFunc("/tasks/execute/", func(w http.ResponseWriter, r *http.Request) {
        // path: /tasks/execute/{id}
        if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
        id := r.URL.Path[len("/tasks/execute/"):]
        go func() {
            if err := orch.ExecutePlan(context.Background(), id); err != nil {
                log.Printf("execute error: %v", err)
            }
        }()
        w.WriteHeader(http.StatusAccepted)
    })

    mux.HandleFunc("/tasks/", func(w http.ResponseWriter, r *http.Request) {
        // path: /tasks/{id}
        if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
        id := r.URL.Path[len("/tasks/"):]
        t, ok := orch.GetTask(id)
        if !ok { http.NotFound(w, r); return }
        respondJSON(w, t)
    })
}

func respondJSON(w http.ResponseWriter, v any) {
    w.Header().Set("Content-Type", "application/json")
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    enc.Encode(v)
}

func genID() string {
    rand.Seed(time.Now().UnixNano())
    return time.Now().Format("20060102150405") + "-" + string('a'+rune(rand.Intn(26)))
}
