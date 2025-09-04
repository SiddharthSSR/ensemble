package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
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

func (c *AnthropicClient) postJSON(ctx context.Context, body any, out any) error {
    b, _ := json.Marshal(body)
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(b))
    req.Header.Set("x-api-key", c.APIKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("content-type", "application/json")
    httpClient := &http.Client{Timeout: 30 * time.Second}
    res, err := httpClient.Do(req)
    if err != nil { return err }
    defer res.Body.Close()
    if res.StatusCode < 200 || res.StatusCode >= 300 {
        var eresp map[string]any
        _ = json.NewDecoder(res.Body).Decode(&eresp)
        return fmt.Errorf("anthropic status %d: %v", res.StatusCode, eresp)
    }
    return json.NewDecoder(res.Body).Decode(out)
}

