package llm

import (
    "os"
    "strings"
)

// NewFromEnv returns a Client based on environment variables.
// Supported providers:
// - LLM_PROVIDER=openai|anthropic|gemini
// - For OpenAI:   OPENAI_API_KEY, optional LLM_MODEL
// - For Anthropic: ANTHROPIC_API_KEY, optional LLM_MODEL
// - For Gemini:    GOOGLE_API_KEY, optional LLM_MODEL
// If nothing is configured, returns a MockClient.
func NewFromEnv() Client {
    prov := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER")))
    switch prov {
    case "openai":
        if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" {
            return &OpenAIClient{APIKey: key, Model: getModelWithDefault("LLM_MODEL", "gpt-4o-mini"), BaseURL: strings.TrimRight(os.Getenv("OPENAI_API_BASE"), "/")} // default lightweight
        }
    case "anthropic":
        if key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); key != "" {
            return &AnthropicClient{APIKey: key, Model: getModelWithDefault("LLM_MODEL", "claude-3-5-sonnet-latest")}
        }
    case "gemini":
        if key := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY")); key != "" {
            // Use lightweight HTTP client to avoid build tags/deps.
            return &GeminiHTTPClient{APIKey: key, Model: getModelWithDefault("LLM_MODEL", "gemini-1.5-flash")}
        }
    }

    // Auto-detect by API key presence if provider not specified
    if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" {
        return &OpenAIClient{APIKey: key, Model: getModelWithDefault("LLM_MODEL", "gpt-4o-mini"), BaseURL: strings.TrimRight(os.Getenv("OPENAI_API_BASE"), "/")}
    }
    if key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); key != "" {
        return &AnthropicClient{APIKey: key, Model: getModelWithDefault("LLM_MODEL", "claude-3-5-sonnet-latest")}
    }
    if key := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY")); key != "" {
        return &GeminiHTTPClient{APIKey: key, Model: getModelWithDefault("LLM_MODEL", "gemini-1.5-flash")}
    }

    return &MockClient{}
}

func getModelWithDefault(envKey, def string) string {
    if v := strings.TrimSpace(os.Getenv(envKey)); v != "" { return v }
    return def
}
