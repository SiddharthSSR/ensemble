package tools

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "time"
)

type HTTPGetTool struct{}

func (h *HTTPGetTool) Name() string { return "http_get" }

func (h *HTTPGetTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    url, _ := inputs["url"].(string)
    if url == "" {
        return nil, "", fmt.Errorf("missing url")
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, "", err
    }
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, "", err
    }
    defer resp.Body.Close()
    b, _ := io.ReadAll(resp.Body)
    return string(b), fmt.Sprintf("status=%d", resp.StatusCode), nil
}

