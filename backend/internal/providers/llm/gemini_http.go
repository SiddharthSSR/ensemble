package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "net/url"
    "os"
    "strings"
)

type GeminiHTTPClient struct {
    APIKey string
    Model  string
}

func (c *GeminiHTTPClient) GeneratePlan(ctx context.Context, prompt string) (string, error) {
    txt, err := c.generateText(ctx, prompt)
    if err != nil { return "", err }
    return txt, nil
}

func (c *GeminiHTTPClient) Verify(ctx context.Context, prompt string, output string) (bool, string, error) {
    full := prompt + "\nOutput to judge:\n" + output
    txt, err := c.generateText(ctx, full)
    if err != nil { return false, "", err }
    return txt != "", txt, nil
}

func (c *GeminiHTTPClient) GenerateText(ctx context.Context, prompt string) (string, error) {
    return c.generateText(ctx, prompt)
}

func (c *GeminiHTTPClient) generateText(ctx context.Context, prompt string) (string, error) {
    endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", url.PathEscape(c.Model), url.QueryEscape(c.APIKey))
    body := map[string]any{
        "contents": []map[string]any{{
            "role":  "user",
            "parts": []map[string]string{{"text": prompt}},
        }},
    }
    b, _ := json.Marshal(body)
    // allow override via GEMINI_API_URL base
    if base := os.Getenv("GEMINI_API_URL"); base != "" {
        endpoint = fmt.Sprintf("%s/models/%s:generateContent?key=%s", strings.TrimRight(base, "/"), url.PathEscape(c.Model), url.QueryEscape(c.APIKey))
    }
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
    req.Header.Set("content-type", "application/json")
    httpClient := &http.Client{Timeout: clientTimeout()}
    res, err := httpClient.Do(req)
    if err != nil { return "", err }
    defer res.Body.Close()
    if res.StatusCode < 200 || res.StatusCode >= 300 {
        var eresp map[string]any
        _ = json.NewDecoder(res.Body).Decode(&eresp)
        return "", fmt.Errorf("gemini status %d: %v", res.StatusCode, eresp)
    }
    var out struct{
        Candidates []struct{
            Content struct{ Parts []struct{ Text string `json:"text"` } `json:"parts"` } `json:"content"`
        } `json:"candidates"`
    }
    if err := json.NewDecoder(res.Body).Decode(&out); err != nil { return "", err }
    if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
        return "", errors.New("no candidates")
    }
    return out.Candidates[0].Content.Parts[0].Text, nil
}
