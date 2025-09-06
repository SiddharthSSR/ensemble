package agents

import (
    "context"
    "strings"
    "os"

    "github.com/example/agent-orchestrator/internal/models"
)

type Planner interface {
    Plan(ctx context.Context, task *models.Task) (*models.Plan, error)
}

// MockPlanner is a simple rule-based planner for MVP.
type MockPlanner struct{}

func (m *MockPlanner) Plan(ctx context.Context, task *models.Task) (*models.Plan, error) {
    q := strings.ToLower(task.Query)
    if b64, ok := task.Context["pdf_data_base64"].(string); ok && b64 != "" {
        // If the query mentions summarize, summarize the PDF content; otherwise answer the query using the PDF as context.
        if strings.Contains(q, "summarize") || strings.Contains(q, "summarise") {
            return &models.Plan{Steps: []*models.Step{
                {
                    ID:          "step1",
                    Description: "Extract text from PDF",
                    Tool:        "pdf_extract",
                    Inputs:      map[string]any{"data_base64": b64},
                    Status:      models.StatusPending,
                },
                {
                    ID:          "step2",
                    Description: "Summarize PDF content",
                    Tool:        "summarize",
                    Inputs:      map[string]any{"text": "{{step:step1.output}}"},
                    Deps:        []string{"step1"},
                    Status:      models.StatusPending,
                },
            }}, nil
        }
        return &models.Plan{Steps: []*models.Step{
            {
                ID:          "step1",
                Description: "Extract text from PDF",
                Tool:        "pdf_extract",
                Inputs:      map[string]any{"data_base64": b64},
                Status:      models.StatusPending,
            },
            {
                ID:          "step2",
                Description: "Answer question using PDF context",
                Tool:        "llm_answer",
                Inputs:      map[string]any{
                    "text": task.Query,
                    "instructions": "Use the following PDF content as context to answer.\n\nContext:\n{{step:step1.output}}",
                },
                Deps:        []string{"step1"},
                Status:      models.StatusPending,
            },
        }}, nil
    }
    // Generic attachments: use first attachment if present
    if atts, ok := task.Context["attachments"].([]any); ok && len(atts) > 0 {
        first, _ := atts[0].(map[string]any)
        if data, ok := first["data_base64"].(string); ok && data != "" {
            fname, _ := first["filename"].(string)
            ctype, _ := first["content_type"].(string)
            lf, lc := strings.ToLower(fname), strings.ToLower(ctype)
            isCSV := strings.HasSuffix(lf, ".csv") || strings.Contains(lc, "csv") || strings.Contains(lc, "ms-excel")
            isJSON := strings.HasSuffix(lf, ".json") || strings.Contains(lc, "json")
            // If summarize requested, summarize the extracted text; else answer using it as context
            if strings.Contains(q, "summarize") || strings.Contains(q, "summarise") {
                if isJSON {
                    return &models.Plan{Steps: []*models.Step{
                        { ID: "step1", Description: "Extract text from file", Tool: pickUnified("file_extract"), Inputs: wrapIfUnified("file_extract", map[string]any{"data_base64": data, "filename": first["filename"], "content_type": first["content_type"]}), Status: models.StatusPending },
                        { ID: "step2", Description: "Pretty-print JSON", Tool: pickUnified("json_pretty"), Inputs: wrapIfUnified("json_pretty", map[string]any{"json": "{{step:step1.output}}"}), Deps: []string{"step1"}, Status: models.StatusPending },
                        { ID: "step3", Description: "Summarize JSON", Tool: pickUnified("summarize"), Inputs: wrapIfUnified("summarize", map[string]any{"text": "{{step:step2.output}}"}), Deps: []string{"step2"}, Status: models.StatusPending },
                    }}, nil
                }
                if isCSV {
                    return &models.Plan{Steps: []*models.Step{
                        { ID: "step1", Description: "Extract text from file", Tool: pickUnified("file_extract"), Inputs: wrapIfUnified("file_extract", map[string]any{"data_base64": data, "filename": first["filename"], "content_type": first["content_type"]}), Status: models.StatusPending },
                        { ID: "step2", Description: "Parse CSV to rows", Tool: pickUnified("csv_parse"), Inputs: wrapIfUnified("csv_parse", map[string]any{"csv": "{{step:step1.output}}"}), Deps: []string{"step1"}, Status: models.StatusPending },
                        { ID: "step3", Description: "Summarize parsed CSV", Tool: pickUnified("summarize"), Inputs: wrapIfUnified("summarize", map[string]any{"text": "{{step:step2.output}}"}), Deps: []string{"step2"}, Status: models.StatusPending },
                    }}, nil
                }
                return &models.Plan{Steps: []*models.Step{
                    { ID: "step1", Description: "Extract text from file", Tool: pickUnified("file_extract"), Inputs: wrapIfUnified("file_extract", map[string]any{"data_base64": data, "filename": first["filename"], "content_type": first["content_type"]}), Status: models.StatusPending },
                    { ID: "step2", Description: "Summarize file content", Tool: pickUnified("summarize"), Inputs: wrapIfUnified("summarize", map[string]any{"text": "{{step:step1.output}}"}), Deps: []string{"step1"}, Status: models.StatusPending },
                }}, nil
            }
            if isJSON {
                return &models.Plan{Steps: []*models.Step{
                    { ID: "step1", Description: "Extract text from file", Tool: pickUnified("file_extract"), Inputs: wrapIfUnified("file_extract", map[string]any{"data_base64": data, "filename": first["filename"], "content_type": first["content_type"]}), Status: models.StatusPending },
                    { ID: "step2", Description: "Pretty-print JSON", Tool: pickUnified("json_pretty"), Inputs: wrapIfUnified("json_pretty", map[string]any{"json": "{{step:step1.output}}"}), Deps: []string{"step1"}, Status: models.StatusPending },
                    { ID: "step3", Description: "Answer using JSON context", Tool: pickUnified("llm_answer"), Inputs: wrapIfUnified("llm_answer", map[string]any{ "text": task.Query, "instructions": "Use the following JSON as context.\n\nContext:\n{{step:step2.output}}" }), Deps: []string{"step2"}, Status: models.StatusPending },
                }}, nil
            }
            if isCSV {
                return &models.Plan{Steps: []*models.Step{
                    { ID: "step1", Description: "Extract text from file", Tool: pickUnified("file_extract"), Inputs: wrapIfUnified("file_extract", map[string]any{"data_base64": data, "filename": first["filename"], "content_type": first["content_type"]}), Status: models.StatusPending },
                    { ID: "step2", Description: "Parse CSV to rows", Tool: pickUnified("csv_parse"), Inputs: wrapIfUnified("csv_parse", map[string]any{"csv": "{{step:step1.output}}"}), Deps: []string{"step1"}, Status: models.StatusPending },
                    { ID: "step3", Description: "Answer using CSV context", Tool: pickUnified("llm_answer"), Inputs: wrapIfUnified("llm_answer", map[string]any{ "text": task.Query, "instructions": "Use the following CSV data (JSON) as context.\n\nContext:\n{{step:step2.output}}" }), Deps: []string{"step2"}, Status: models.StatusPending },
                }}, nil
            }
            return &models.Plan{Steps: []*models.Step{
                { ID: "step1", Description: "Extract text from file", Tool: pickUnified("file_extract"), Inputs: wrapIfUnified("file_extract", map[string]any{"data_base64": data, "filename": first["filename"], "content_type": first["content_type"]}), Status: models.StatusPending },
                { ID: "step2", Description: "Answer question using file context", Tool: pickUnified("llm_answer"), Inputs: wrapIfUnified("llm_answer", map[string]any{ "text": task.Query, "instructions": "Use the following file content as context to answer.\n\nContext:\n{{step:step1.output}}" }), Deps: []string{"step1"}, Status: models.StatusPending },
            }}, nil
        }
    }
    // Prefer richer defaults: URL -> http_get -> html_to_text -> summarize, else llm_answer
    if strings.Contains(q, "http") || strings.HasPrefix(q, "http://") || strings.HasPrefix(q, "https://") {
        return &models.Plan{Steps: []*models.Step{
            {
                ID:          "step1",
                Description: "HTTP GET a URL",
                Tool:        "http_get",
                Inputs:      map[string]any{"url": task.Query},
                Status:      models.StatusPending,
            },
            {
                ID:          "step2",
                Description: "Convert HTML to text",
                Tool:        "html_to_text",
                Inputs:      map[string]any{"html": "{{step:step1.output}}"},
                Deps:        []string{"step1"},
                Status:      models.StatusPending,
            },
            {
                ID:          "step3",
                Description: "Summarize content (chunked)",
                Tool:        "summarize_chunked",
                Inputs:      map[string]any{"text": "{{step:step2.output}}"},
                Deps:        []string{"step2"},
                Status:      models.StatusPending,
            },
        }}, nil
    }
    return &models.Plan{Steps: []*models.Step{{
        ID:          "step1",
        Description: "Answer with LLM",
        Tool:        "llm_answer",
        Inputs:      map[string]any{"text": task.Query},
        Status:      models.StatusPending,
    }}}, nil
}

// Helpers to support unified call_tool in MockPlanner when env USE_UNIFIED_TOOL=1
func pickUnified(tool string) string {
    if strings.TrimSpace(os.Getenv("USE_UNIFIED_TOOL")) == "1" { return "call_tool" }
    return tool
}
func wrapIfUnified(tool string, m map[string]any) map[string]any {
    if strings.TrimSpace(os.Getenv("USE_UNIFIED_TOOL")) == "1" {
        return map[string]any{"tool": tool, "inputs": m}
    }
    return m
}
