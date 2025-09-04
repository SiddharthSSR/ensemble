package api

import (
    "context"
    "encoding/json"
    "log"
    "math/rand"
    "net/http"
    "time"

    "github.com/example/agent-orchestrator/internal/agents"
    "github.com/example/agent-orchestrator/internal/models"
    "github.com/example/agent-orchestrator/internal/orchestrator"
    "github.com/example/agent-orchestrator/internal/tools"
)

var orch *orchestrator.Orchestrator

func init() {
    // Wire default components for MVP
    reg := tools.NewRegistry()
    reg.Register(&tools.EchoTool())
    reg.Register(&tools.HTTPGetTool())
    orch = orchestrator.New(&agents.MockPlanner{}, &agents.ToolExecutor{Registry: reg}, &agents.SimpleVerifier{})
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

