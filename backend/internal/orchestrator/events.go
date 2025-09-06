package orchestrator

import (
    "encoding/json"
    "sync"
    "time"
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

    tokMu   sync.Mutex
    tokBuf  map[string]map[string]string // taskID -> stepID -> buffered chunk(s)
    tokTick map[string]chan struct{}     // taskID -> stop channel
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

// TokenAppender returns a function to buffer token chunks per step for a task and
// periodically flush them as coalesced 'token' events (100ms cadence).
func (h *Hub) TokenAppender(taskID string) func(stepID, chunk string) {
    h.tokMu.Lock()
    if h.tokBuf == nil { h.tokBuf = map[string]map[string]string{} }
    if h.tokTick == nil { h.tokTick = map[string]chan struct{}{} }
    if _, ok := h.tokBuf[taskID]; !ok { h.tokBuf[taskID] = map[string]string{} }
    if _, ok := h.tokTick[taskID]; !ok {
        stop := make(chan struct{})
        h.tokTick[taskID] = stop
        go h.flushLoop(taskID, stop)
    }
    h.tokMu.Unlock()
    return func(stepID, chunk string) {
        if chunk == "" || stepID == "" { return }
        h.tokMu.Lock()
        if _, ok := h.tokBuf[taskID]; !ok { h.tokBuf[taskID] = map[string]string{} }
        h.tokBuf[taskID][stepID] = h.tokBuf[taskID][stepID] + chunk
        h.tokMu.Unlock()
    }
}

func (h *Hub) flushLoop(taskID string, stop <-chan struct{}) {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-stop:
            return
        case <-ticker.C:
            h.tokMu.Lock()
            buf := h.tokBuf[taskID]
            if len(buf) == 0 { h.tokMu.Unlock(); continue }
            // copy then clear
            payloads := make(map[string]string, len(buf))
            for sid, s := range buf { if s != "" { payloads[sid] = s } }
            for sid := range buf { delete(buf, sid) }
            h.tokMu.Unlock()
            for sid, chunk := range payloads {
                h.Publish(taskID, Event{Event: "token", TaskID: taskID, Payload: map[string]any{"step_id": sid, "chunk": chunk}})
            }
        }
    }
}

// StopTokenAppender stops the coalescer for a task and flushes remaining chunks.
func (h *Hub) StopTokenAppender(taskID string) {
    h.tokMu.Lock()
    if ch, ok := h.tokTick[taskID]; ok { close(ch); delete(h.tokTick, taskID) }
    buf := h.tokBuf[taskID]
    delete(h.tokBuf, taskID)
    h.tokMu.Unlock()
    // flush leftovers synchronously
    for sid, chunk := range buf {
        if chunk == "" { continue }
        h.Publish(taskID, Event{Event: "token", TaskID: taskID, Payload: map[string]any{"step_id": sid, "chunk": chunk}})
    }
}
