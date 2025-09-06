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
    // limit body to avoid huge transfers
    max := envInt("HTTP_GET_MAX_BYTES", 2<<20)
    lr := io.LimitedReader{R: resp.Body, N: int64(max)}
    b, _ := io.ReadAll(&lr)
    logs := fmt.Sprintf("status=%d", resp.StatusCode)
    if lr.N == 0 { logs += " truncated=true" }
    return string(b), logs, nil
}
