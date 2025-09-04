package tools

import (
    "context"
    "strings"

    "golang.org/x/net/html"
)

type HTMLToTextTool struct{}

func (t *HTMLToTextTool) Name() string { return "html_to_text" }

func (t *HTMLToTextTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    htmlStr, _ := inputs["html"].(string)
    if htmlStr == "" { return "", "", nil }
    node, err := html.Parse(strings.NewReader(htmlStr))
    if err != nil { return "", "", err }
    var b strings.Builder
    extractText(node, &b, false)
    out := strings.TrimSpace(compactWhitespace(b.String()))
    return out, "", nil
}

func extractText(n *html.Node, b *strings.Builder, inHidden bool) {
    if n.Type == html.ElementNode {
        // skip script/style/noscript
        switch strings.ToLower(n.Data) {
        case "script", "style", "noscript":
            inHidden = true
        case "br", "p", "div", "li", "tr":
            b.WriteString("\n")
        }
    }
    if !inHidden && n.Type == html.TextNode {
        b.WriteString(n.Data)
    }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        extractText(c, b, inHidden)
    }
}

func compactWhitespace(s string) string {
    // collapse multiple spaces/newlines
    s = strings.ReplaceAll(s, "\t", " ")
    s = strings.ReplaceAll(s, "\r", " ")
    lines := strings.Split(s, "\n")
    for i, ln := range lines {
        lines[i] = strings.Join(strings.Fields(ln), " ")
    }
    // remove empty lines
    var out []string
    for _, ln := range lines {
        if strings.TrimSpace(ln) != "" {
            out = append(out, ln)
        }
    }
    return strings.Join(out, "\n")
}

