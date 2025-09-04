package llm

import (
    "context"
    "strings"
)

// MockClient is used when no real provider is configured.
type MockClient struct{}

func (m *MockClient) GeneratePlan(ctx context.Context, prompt string) (string, error) {
    p := strings.ToLower(prompt)
    if strings.Contains(p, "http") || strings.Contains(p, "url") {
        return `[{"id":"step1","description":"HTTP GET a URL","tool":"http_get","inputs":{"url":"<from-query>"}}]`, nil
    }
    return `[{"id":"step1","description":"Echo the query","tool":"echo","inputs":{"text":"<from-query>"}}]`, nil
}

func (m *MockClient) Verify(ctx context.Context, prompt string, output string) (bool, string, error) {
    return strings.TrimSpace(output) != "", "ok", nil
}

func (m *MockClient) GenerateText(ctx context.Context, prompt string) (string, error) {
    // Return a short canned response for testing
    return "(mock) " + truncate(strings.TrimSpace(prompt), 200), nil
}

func truncate(s string, n int) string {
    if len(s) <= n { return s }
    return s[:n] + "..."
}
