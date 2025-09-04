package main

import (
    "log"
    "net/http"
    "os"

    "github.com/example/agent-orchestrator/internal/api"
)

func main() {
    addr := ":8080"
    if v := os.Getenv("PORT"); v != "" {
        addr = ":" + v
    }

    mux := http.NewServeMux()
    api.RegisterRoutes(mux)

    log.Printf("server listening on %s", addr)
    if err := http.ListenAndServe(addr, cors(mux)); err != nil {
        log.Fatal(err)
    }
}

// simple CORS middleware for local dev
func cors(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        next.ServeHTTP(w, r)
    })
}

