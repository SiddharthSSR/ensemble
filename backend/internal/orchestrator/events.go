package orchestrator

import (
    "encoding/json"
    "sync"
)

// Event is a generic SSE payload wrapper.
type Event struct {
    Event   string      `json:"event"`
    TaskID  string      `json:"task_id"`
    Payload interface{} `json:"payload,omitempty"`
}

type subscriber chan []byte

type Hub struct {
    mu    sync.RWMutex
    subs  map[string]map[subscriber]struct{} // taskID -> set of subscribers
}

func NewHub() *Hub { return &Hub{subs: map[string]map[subscriber]struct{}{}} }

func (h *Hub) Subscribe(taskID string) (subscriber, func()) {
    ch := make(subscriber, 16)
    h.mu.Lock()
    set := h.subs[taskID]
    if set == nil { set = map[subscriber]struct{}{}; h.subs[taskID] = set }
    set[ch] = struct{}{}
    h.mu.Unlock()
    unsubscribe := func() {
        h.mu.Lock()
        if set, ok := h.subs[taskID]; ok {
            delete(set, ch)
            if len(set) == 0 { delete(h.subs, taskID) }
        }
        close(ch)
        h.mu.Unlock()
    }
    return ch, unsubscribe
}

func (h *Hub) Publish(taskID string, ev Event) {
    b, _ := json.Marshal(ev)
    h.mu.RLock()
    set := h.subs[taskID]
    for ch := range set {
        // non-blocking send
        select { case ch <- b: default: }
    }
    h.mu.RUnlock()
}

