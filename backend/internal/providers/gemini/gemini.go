package gemini

// Placeholder for Gemini client integration.
// To keep the MVP buildable without external deps, this file exposes
// a minimal interface and no-op implementation you can replace later
// with github.com/google/generative-ai-go/genai.

import (
    "context"
)

type Client interface {
    // Plan should accept a prompt and return a structured plan proposal.
    GeneratePlan(ctx context.Context, prompt string) (string, error)
    // Verify should validate a given result against acceptance criteria.
    Verify(ctx context.Context, prompt string, output string) (bool, string, error)
}

// MockClient is used in development when GOOGLE_API_KEY is not set.
type MockClient struct{}

func (m *MockClient) GeneratePlan(ctx context.Context, prompt string) (string, error) {
    return "[{\"id\":\"step1\",\"tool\":\"echo\",\"description\":\"Echo input\"}]", nil
}

func (m *MockClient) Verify(ctx context.Context, prompt string, output string) (bool, string, error) {
    return output != "", "ok", nil
}

