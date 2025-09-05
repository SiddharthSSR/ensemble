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

type AnthropicClient struct {
    APIKey string
    Model  string
}

func (c *AnthropicClient) GeneratePlan(ctx context.Context, prompt string) (string, error) {
    body := map[string]any{
        "model": c.Model,
        "max_tokens": 1024,
        "messages": []map[string]any{{
            "role": "user",
            "content": []map[string]string{{"type": "text", "text": prompt}},
        }},
    }
    var resp struct{ Content []struct{ Text string `json:"text"` } `json:"content"` }
    if err := c.postJSON(ctx, body, &resp); err != nil { return "", err }
    if len(resp.Content) == 0 { return "", errors.New("no content") }
    return resp.Content[0].Text, nil
}

func (c *AnthropicClient) Verify(ctx context.Context, prompt string, output string) (bool, string, error) {
    full := fmt.Sprintf("%s\nOutput to judge:\n%s", prompt, output)
    body := map[string]any{
        "model": c.Model,
        "max_tokens": 1024,
        "messages": []map[string]any{{
            "role": "user",
            "content": []map[string]string{{"type": "text", "text": full}},
        }},
    }
    var resp struct{ Content []struct{ Text string `json:"text"` } `json:"content"` }
    if err := c.postJSON(ctx, body, &resp); err != nil { return false, "", err }
    if len(resp.Content) == 0 { return false, "", errors.New("no content") }
    txt := resp.Content[0].Text
    return txt != "", txt, nil
}

func (c *AnthropicClient) GenerateText(ctx context.Context, prompt string) (string, error) {
    body := map[string]any{
        "model": c.Model,
        "max_tokens": 1024,
        "messages": []map[string]any{{
            "role": "user",
            "content": []map[string]string{{"type": "text", "text": prompt}},
        }},
    }
    var resp struct{ Content []struct{ Text string `json:"text"` } `json:"content"` }
    if err := c.postJSON(ctx, body, &resp); err != nil { return "", err }
    if len(resp.Content) == 0 { return "", errors.New("no content") }
    return resp.Content[0].Text, nil
}

func (c *AnthropicClient) GenerateTextStream(ctx context.Context, prompt string, onDelta func(chunk string) error) error {
    // Fallback to non-streaming for now
    txt, err := c.GenerateText(ctx, prompt)
    if err != nil { return err }
    if err := onDelta(txt); err != nil { return err }
    return nil
}

func (c *AnthropicClient) postJSON(ctx context.Context, body any, out any) error {
    b, _ := json.Marshal(body)
    url := os.Getenv("ANTHROPIC_API_URL")
    if url == "" { url = "https://api.anthropic.com/v1/messages" }
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
    req.Header.Set("x-api-key", c.APIKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("content-type", "application/json")
    httpClient := &http.Client{Timeout: clientTimeout()}
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
        lastErr = fmt.Errorf("anthropic status %d: %v", res.StatusCode, eresp)
        if res.StatusCode == 408 || res.StatusCode == 429 || (res.StatusCode >= 500 && res.StatusCode <= 599) {
            time.Sleep(backoff(attempt));
            continue
        }
        return lastErr
    }
    return lastErr
}
