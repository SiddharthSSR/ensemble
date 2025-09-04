package tools

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "time"
)

type HTTPPostJSONTool struct{}

func (h *HTTPPostJSONTool) Name() string { return "http_post_json" }

func (h *HTTPPostJSONTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    rawURL, _ := inputs["url"].(string)
    if rawURL == "" { return nil, "", fmt.Errorf("missing url") }
    u, err := url.Parse(rawURL)
    if err != nil { return nil, "", fmt.Errorf("invalid url: %w", err) }
    if u.Scheme != "http" && u.Scheme != "https" {
        return nil, "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
    }

    // payload
    var bodyBytes []byte
    if s, ok := inputs["json"].(string); ok && s != "" {
        bodyBytes = []byte(s)
    } else {
        bodyBytes, err = json.Marshal(inputs["json"]) // may be nil -> "null"
        if err != nil { return nil, "", fmt.Errorf("marshal json: %w", err) }
    }

    // headers
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(bodyBytes))
    if err != nil { return nil, "", err }
    req.Header.Set("Content-Type", "application/json")
    if hv, ok := inputs["headers"].(map[string]any); ok {
        for k, v := range hv {
            if vs, ok := v.(string); ok { req.Header.Set(k, vs) }
        }
    }

    // timeout
    timeout := 10 * time.Second
    if tv, ok := inputs["timeout_ms"].(float64); ok && tv > 0 { timeout = time.Duration(int(tv)) * time.Millisecond }
    client := &http.Client{Timeout: timeout}

    resp, err := client.Do(req)
    if err != nil { return nil, "", err }
    defer resp.Body.Close()

    // limit body to 2MB to avoid memory blowup
    const max = 2 << 20
    lr := io.LimitedReader{R: resp.Body, N: max}
    b, _ := io.ReadAll(&lr)
    logs := fmt.Sprintf("status=%d content_type=%s", resp.StatusCode, resp.Header.Get("Content-Type"))
    return string(b), logs, nil
}

