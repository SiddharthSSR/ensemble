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
    "fmt"
    "strings"
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
    reg.Register(&tools.ExtractLinksTool{})
    reg.Register(&tools.CSVParseTool{})
    reg.Register(&tools.RegexExtractTool{})
    reg.Register(&tools.HTTPPostJSONTool{})
    reg.Register(&tools.PDFExtractTool{})
    reg.Register(&tools.FileExtractTool{})
    reg.Register(&tools.JSONPrettyTool{})
    // Unified delegator tool
    reg.Register(&tools.CallTool{Registry: reg})
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

    mux.HandleFunc("/debug/llm", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
        client := llm.NewFromEnv()
        // derive provider/model best-effort
        p := fmt.Sprintf("%T", client)
        model := ""
        switch c := client.(type) {
        case *llm.OpenAIClient:
            model = c.Model
            p = "openai"
        case *llm.AnthropicClient:
            model = c.Model
            p = "anthropic"
        case *llm.GeminiHTTPClient:
            model = c.Model
            p = "gemini"
        default:
            if strings.Contains(p, "MockClient") { p = "mock" }
        }
        // do a quick test
        ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
        defer cancel()
        _, err := client.GenerateText(ctx, "ping")
        resp := map[string]any{"provider": p, "model": model, "ok": err == nil}
        if err != nil { resp["error"] = err.Error() }
        respondJSON(w, resp)
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
        // Handle both JSON task fetch and SSE stream under /tasks/{id} and /tasks/{id}/events
        if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
        if strings.HasSuffix(r.URL.Path, "/events") {
            id := strings.TrimSuffix(r.URL.Path[len("/tasks/"):], "/events")
            // set SSE headers
            w.Header().Set("Content-Type", "text/event-stream")
            w.Header().Set("Cache-Control", "no-cache")
            w.Header().Set("Connection", "keep-alive")
            flusher, ok := w.(http.Flusher)
            if !ok { http.Error(w, "stream unsupported", http.StatusInternalServerError); return }
            ch, unsub := orch.Subscribe(id)
            defer unsub()
            if t, ok := orch.GetTask(id); ok {
                b, _ := json.Marshal(t)
                writeSSE(w, "snapshot", b)
                flusher.Flush()
            }
            ticker := time.NewTicker(20 * time.Second)
            defer ticker.Stop()
            notify := r.Context().Done()
            for {
                select {
                case <-notify:
                    return
                case msg, ok := <-ch:
                    if !ok { return }
                    writeSSE(w, "update", msg)
                    flusher.Flush()
                case <-ticker.C:
                    w.Write([]byte(": ping\n\n"))
                    flusher.Flush()
                }
            }
        }
        // JSON task fetch: /tasks/{id}
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

func writeSSE(w http.ResponseWriter, event string, data []byte) {
    w.Write([]byte("event: "+event+"\n"))
    // ensure single-line data chunks by JSON encoding beforehand
    w.Write([]byte("data: "))
    w.Write(data)
    w.Write([]byte("\n\n"))
}

func genID() string {
    rand.Seed(time.Now().UnixNano())
    return time.Now().Format("20060102150405") + "-" + string('a'+rune(rand.Intn(26)))
}
