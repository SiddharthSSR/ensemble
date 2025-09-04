package gemini

// Placeholder for Gemini client integration.
// Default build provides a mock client and factory returning it.
// A real client is provided under build tag `gemini`.

import (
    "context"
    "os"
    "strings"
)

type Client interface {
    GeneratePlan(ctx context.Context, prompt string) (string, error)
    Verify(ctx context.Context, prompt string, output string) (bool, string, error)
}

// MockClient is used in development when GOOGLE_API_KEY is not set.
type MockClient struct{}

func (m *MockClient) GeneratePlan(ctx context.Context, prompt string) (string, error) {
    // naive heuristic: if prompt suggests http/url, return http_get, otherwise echo
    p := strings.ToLower(prompt)
    if strings.Contains(p, "http") || strings.Contains(p, "url") {
        return `[{"id":"step1","description":"HTTP GET a URL","tool":"http_get","inputs":{"url":"<from-query>"}}]`, nil
    }
    return `[{"id":"step1","description":"Echo the query","tool":"echo","inputs":{"text":"<from-query>"}}]`, nil
}

func (m *MockClient) Verify(ctx context.Context, prompt string, output string) (bool, string, error) {
    return strings.TrimSpace(output) != "", "ok", nil
}

// NewFromEnv returns a Client. Default (no build tag) returns a MockClient.
func NewFromEnv() Client {
    _ = os.Getenv("GOOGLE_API_KEY") // ignored in mock
    return &MockClient{}
}
