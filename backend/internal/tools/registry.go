package tools

import "context"

type Tool interface {
    Name() string
    Execute(ctx context.Context, inputs map[string]any) (output any, logs string, err error)
}

type Registry struct {
    tools map[string]Tool
}

func NewRegistry() *Registry {
    return &Registry{tools: map[string]Tool{}}
}

func (r *Registry) Register(t Tool) {
    r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
    t, ok := r.tools[name]
    return t, ok
}

