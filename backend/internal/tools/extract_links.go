package tools

import (
    "context"
    "net/url"
    "strings"

    "golang.org/x/net/html"
)

// ExtractLinksTool parses an HTML string and returns links with text.
// Inputs:
// - html: string (required)
// - base_url: string (optional; used to resolve relative hrefs)
// - max: number (optional; default 50)
// Output: []{ href, text }
type ExtractLinksTool struct{}

func (t *ExtractLinksTool) Name() string { return "extract_links" }

func (t *ExtractLinksTool) Execute(ctx context.Context, inputs map[string]any) (any, string, error) {
    htmlStr, _ := inputs["html"].(string)
    if strings.TrimSpace(htmlStr) == "" { return []map[string]string{}, "", nil }
    max := 50
    if v, ok := inputs["max"].(float64); ok && v > 0 { max = int(v) }

    var base *url.URL
    if b, ok := inputs["base_url"].(string); ok && b != "" {
        if u, err := url.Parse(b); err == nil { base = u }
    }

    root, err := html.Parse(strings.NewReader(htmlStr))
    if err != nil { return nil, "", err }
    var out []map[string]string
    var walk func(*html.Node)
    walk = func(n *html.Node) {
        if n == nil || len(out) >= max { return }
        if n.Type == html.ElementNode && strings.EqualFold(n.Data, "a") {
            var href, text string
            for _, a := range n.Attr {
                if strings.EqualFold(a.Key, "href") { href = strings.TrimSpace(a.Val); break }
            }
            text = strings.TrimSpace(nodeText(n))
            if href != "" {
                if base != nil {
                    if u, err := url.Parse(href); err == nil { href = base.ResolveReference(u).String() }
                }
                out = append(out, map[string]string{"href": href, "text": text})
            }
        }
        for c := n.FirstChild; c != nil && len(out) < max; c = c.NextSibling { walk(c) }
    }
    walk(root)
    return out, "", nil
}

func nodeText(n *html.Node) string {
    if n == nil { return "" }
    var b strings.Builder
    var rec func(*html.Node)
    rec = func(x *html.Node) {
        if x.Type == html.TextNode { b.WriteString(x.Data) }
        for c := x.FirstChild; c != nil; c = c.NextSibling { rec(c) }
    }
    rec(n)
    // compact whitespace
    parts := strings.Fields(b.String())
    return strings.Join(parts, " ")
}

