package tools

import (
    "context"
    "fmt"
    "sync"

    "github.com/example/agent-orchestrator/internal/providers/llm"
)

// SummarizeChunkedTool splits large text into chunks, summarizes each (bounded concurrency),
// then reduces into a concise overall summary. Streams only the reduce phase.
// Inputs:
// - text: string (required)
// - chunk_chars: number (optional; default 8000)
// - overlap_chars: number (optional; default 400)
// - max_parallel: number (optional; default 3)
// - reduce_instructions: string (optional)
type SummarizeChunkedTool struct{ Client llm.Client }

func (t *SummarizeChunkedTool) Name() string { return "summarize_chunked" }

func (t *SummarizeChunkedTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    text, _ := inputs["text"].(string)
    if text == "" { return nil, "", fmt.Errorf("missing text") }
    chunk := getInt(inputs, "chunk_chars", envInt("CHUNK_CHARS", 8000))
    overlap := getInt(inputs, "overlap_chars", envInt("CHUNK_OVERLAP", 400))
    if chunk < 1000 { chunk = 1000 }
    if overlap < 0 { overlap = 0 }
    parts := splitChunks(text, chunk, overlap)
    if len(parts) == 1 {
        // small text, fallback to single summarize
        return (&SummarizeTool{Client: t.Client}).Execute(ctx, map[string]any{"text": text})
    }
    // map phase
    type item struct{ i int; sum string; err error }
    out := make([]string, len(parts))
    sem := make(chan struct{}, getInt(inputs, "max_parallel", envInt("CHUNK_MAX_PAR", 3)))
    var wg sync.WaitGroup
    errs := make([]error, len(parts))
    for i, p := range parts {
        wg.Add(1); sem <- struct{}{}
        go func(i int, p string) {
            defer wg.Done(); defer func(){ <-sem }()
            prompt := fmt.Sprintf("Summarize this section into 3-5 concise bullets focusing on key facts.\n\nSection %d/%d:\n%s", i+1, len(parts), p)
            s, err := t.Client.GenerateText(ctx, prompt)
            if err != nil { errs[i] = err; return }
            out[i] = s
        }(i, p)
    }
    wg.Wait()
    for _, e := range errs { if e != nil { return nil, "", e } }
    // reduce phase (stream)
    reduceInst, _ := inputs["reduce_instructions"].(string)
    if reduceInst == "" {
        reduceInst = "Combine the following section summaries into a single clear summary (bullets or short paragraphs). Avoid repetition; preserve critical details."
    }
    var combined string
    for i, s := range out {
        combined += fmt.Sprintf("\n\n[Section %d]\n%s", i+1, s)
    }
    prompt := reduceInst + "\n\nSummaries:" + combined
    if cb, ok := ctx.Value(CtxTokenCallbackKey).(TokenCallback); ok && cb != nil {
        var acc string
        if err := t.Client.GenerateTextStream(ctx, prompt, func(chunk string) error { acc += chunk; cb(chunk); return nil }); err != nil { return nil, "", err }
        return acc, "", nil
    }
    final, err := t.Client.GenerateText(ctx, prompt)
    if err != nil { return nil, "", err }
    return final, "", nil
}

func splitChunks(s string, size, overlap int) []string {
    if size <= 0 { return []string{s} }
    var out []string
    for start := 0; start < len(s); {
        end := start + size
        if end > len(s) { end = len(s) }
        out = append(out, s[start:end])
        if end == len(s) { break }
        next := end - overlap
        if next <= start { next = end }
        start = next
    }
    return out
}

