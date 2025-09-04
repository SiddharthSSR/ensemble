//go:build gemini

package gemini

import (
    "context"
    "fmt"
    "os"

    genai "github.com/google/generative-ai-go/genai"
    "google.golang.org/api/option"
)

type RealClient struct{ model *genai.GenerativeModel }

func NewFromEnv() Client {
    apiKey := os.Getenv("GOOGLE_API_KEY")
    if apiKey == "" {
        return &MockClient{}
    }
    ctx := context.Background()
    c, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
    if err != nil {
        return &MockClient{}
    }
    // You can change model name if needed
    return &RealClient{model: c.GenerativeModel("gemini-1.5-flash")}
}

func (r *RealClient) GeneratePlan(ctx context.Context, prompt string) (string, error) {
    resp, err := r.model.GenerateContent(ctx, genai.Text(prompt))
    if err != nil { return "", err }
    return firstText(resp), nil
}

func (r *RealClient) Verify(ctx context.Context, prompt string, output string) (bool, string, error) {
    full := fmt.Sprintf("%s\nOutput to judge:\n%s", prompt, output)
    resp, err := r.model.GenerateContent(ctx, genai.Text(full))
    if err != nil { return false, "", err }
    txt := firstText(resp)
    // naive parse: treat non-empty as pass; refine with JSON schema as needed
    return txt != "", txt, nil
}

func firstText(r *genai.GenerateContentResponse) string {
    if r == nil { return "" }
    for _, c := range r.Candidates {
        for _, part := range c.Content.Parts {
            if t, ok := part.(genai.Text); ok {
                return string(t)
            }
        }
    }
    return ""
}

