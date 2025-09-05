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
    // GenerateTextStream streams text chunks to onDelta; implementers may fall back to a single final chunk.
    GenerateTextStream(ctx context.Context, prompt string, onDelta func(chunk string) error) error
}
