package models

import (
    "time"
)

type Status string

const (
    StatusPending  Status = "PENDING"
    StatusPlanned  Status = "PLANNED"
    StatusRunning  Status = "RUNNING"
    StatusSuccess  Status = "SUCCESS"
    StatusFailed   Status = "FAILED"
)

type Task struct {
    ID        string            `json:"id"`
    Query     string            `json:"query"`
    Context   map[string]any    `json:"context,omitempty"`
    Status    Status            `json:"status"`
    Plan      *Plan             `json:"plan,omitempty"`
    Results   []*Result         `json:"results,omitempty"`
    CreatedAt time.Time         `json:"created_at"`
    UpdatedAt time.Time         `json:"updated_at"`
}

type Plan struct {
    Steps []*Step `json:"steps"`
}

type Step struct {
    ID          string         `json:"id"`
    Description string         `json:"description"`
    Deps        []string       `json:"deps,omitempty"`
    Tool        string         `json:"tool"`
    Inputs      map[string]any `json:"inputs,omitempty"`
    Status      Status         `json:"status"`
}

type Result struct {
    StepID   string `json:"step_id"`
    Output   any    `json:"output,omitempty"`
    Logs     string `json:"logs,omitempty"`
    Verified bool   `json:"verified"`
    Error    string `json:"error,omitempty"`
    Retries  int    `json:"retries"`
}
