package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "os"
    "time"
)

type OpenAIClient struct {
    APIKey string
    Model  string
    BaseURL string
}

func (c *OpenAIClient) GeneratePlan(ctx context.Context, prompt string) (string, error) {
    // Use Chat Completions for broad compatibility
    body := map[string]any{
        "model": c.Model,
        "messages": []map[string]string{{"role": "user", "content": prompt}},
        "temperature": 0.2,
    }
    var resp struct{
        Choices []struct{ Message struct{ Content string `json:"content"` } `json:"message"` } `json:"choices"`
    }
    if err := c.postJSON(ctx, c.endpoint("/v1/chat/completions"), body, &resp); err != nil {
        return "", err
    }
    if len(resp.Choices) == 0 { return "", errors.New("no choices") }
    return resp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) Verify(ctx context.Context, prompt string, output string) (bool, string, error) {
    full := fmt.Sprintf("%s\nOutput to judge:\n%s", prompt, output)
    body := map[string]any{
        "model": c.Model,
        "messages": []map[string]string{{"role": "user", "content": full}},
        "temperature": 0,
    }
    var resp struct{
        Choices []struct{ Message struct{ Content string `json:"content"` } `json:"message"` } `json:"choices"`
    }
    if err := c.postJSON(ctx, c.endpoint("/v1/chat/completions"), body, &resp); err != nil {
        return false, "", err
    }
    if len(resp.Choices) == 0 { return false, "", errors.New("no choices") }
    content := resp.Choices[0].Message.Content
    // Best-effort: treat non-empty as pass; caller can parse JSON if desired
    ok := content != ""
    return ok, content, nil
}

func (c *OpenAIClient) GenerateText(ctx context.Context, prompt string) (string, error) {
    body := map[string]any{
        "model": c.Model,
        "messages": []map[string]string{{"role": "user", "content": prompt}},
        "temperature": 0.3,
    }
    var resp struct{
        Choices []struct{ Message struct{ Content string `json:"content"` } `json:"message"` } `json:"choices"`
    }
    if err := c.postJSON(ctx, c.endpoint("/v1/chat/completions"), body, &resp); err != nil {
        return "", err
    }
    if len(resp.Choices) == 0 { return "", errors.New("no choices") }
    return resp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) postJSON(ctx context.Context, url string, body any, out any) error {
    b, _ := json.Marshal(body)
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
    req.Header.Set("Authorization", "Bearer "+c.APIKey)
    req.Header.Set("Content-Type", "application/json")
    timeout := clientTimeout()
    httpClient := &http.Client{Timeout: timeout}
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        res, err := httpClient.Do(req.Clone(ctx))
        if err != nil { lastErr = err; if isTimeout(err) { time.Sleep(backoff(attempt)); continue }; return err }
        defer res.Body.Close()
        if res.StatusCode >= 200 && res.StatusCode < 300 {
            return json.NewDecoder(res.Body).Decode(out)
        }
        var eresp map[string]any
        _ = json.NewDecoder(res.Body).Decode(&eresp)
        lastErr = fmt.Errorf("openai status %d: %v", res.StatusCode, eresp)
        if res.StatusCode == 408 || res.StatusCode == 429 || (res.StatusCode >= 500 && res.StatusCode <= 599) {
            time.Sleep(backoff(attempt));
            continue
        }
        return lastErr
    }
    return lastErr
}

func (c *OpenAIClient) endpoint(path string) string {
    base := c.BaseURL
    if base == "" { base = os.Getenv("OPENAI_API_BASE") }
    if base == "" { base = "https://api.openai.com" }
    return base + path
}

func clientTimeout() time.Duration {
    if v := os.Getenv("LLM_HTTP_TIMEOUT_MS"); v != "" {
        if ms, err := time.ParseDuration(v+"ms"); err == nil { return ms }
    }
    return 45 * time.Second
}

func isTimeout(err error) bool {
    type timeout interface{ Timeout() bool }
    if te, ok := err.(timeout); ok { return te.Timeout() }
    return false
}

func backoff(i int) time.Duration {
    return time.Duration(500*(1<<i)) * time.Millisecond
}
