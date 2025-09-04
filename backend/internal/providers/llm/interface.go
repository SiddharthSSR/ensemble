package llm

import (
    "context"
)

// Client is a minimal interface used by planner and verifier.
// Any provider implementation should satisfy this.
type Client interface {
    GeneratePlan(ctx context.Context, prompt string) (string, error)
    Verify(ctx context.Context, prompt string, output string) (bool, string, error)
    GenerateText(ctx context.Context, prompt string) (string, error)
}
