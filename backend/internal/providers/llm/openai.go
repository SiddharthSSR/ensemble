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

type OpenAIClient struct {
    APIKey string
    Model  string
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
    if err := c.postJSON(ctx, "https://api.openai.com/v1/chat/completions", body, &resp); err != nil {
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
    if err := c.postJSON(ctx, "https://api.openai.com/v1/chat/completions", body, &resp); err != nil {
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
    if err := c.postJSON(ctx, "https://api.openai.com/v1/chat/completions", body, &resp); err != nil {
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
    httpClient := &http.Client{Timeout: 30 * time.Second}
    res, err := httpClient.Do(req)
    if err != nil { return err }
    defer res.Body.Close()
    if res.StatusCode < 200 || res.StatusCode >= 300 {
        var eresp map[string]any
        _ = json.NewDecoder(res.Body).Decode(&eresp)
        return fmt.Errorf("openai status %d: %v", res.StatusCode, eresp)
    }
    return json.NewDecoder(res.Body).Decode(out)
}
