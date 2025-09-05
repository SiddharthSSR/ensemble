import { useEffect, useMemo, useState } from 'react'

type Task = {
  id: string
  query: string
  status: string
  plan?: { steps: Step[] }
  results?: Result[]
  created_at?: string
  updated_at?: string
}
type Step = { id: string; description: string; tool: string; status: string }
type Result = { step_id: string; output?: any; logs?: string; verified: boolean; error?: string }

const API_BASE = 'http://localhost:8080'
const API = (path: string) => `${API_BASE}${path}`

export default function App() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<Task | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [es, setEs] = useState<EventSource | null>(null)
  const [busy, setBusy] = useState(false)
  const [llmInfo, setLlmInfo] = useState<any | null>(null)

  async function refresh() {
    const res = await fetch(API('/tasks'))
    const data = await res.json()
    setTasks(data)
    if (selected) {
      const sres = await fetch(API(`/tasks/${selected.id}`))
      setSelected(await sres.json())
    }
  }

  useEffect(() => { refresh(); fetchLLM(); }, [])
  useEffect(() => {
    if (!autoRefresh) return
    const t = setInterval(refresh, 1500)
    return () => clearInterval(t)
  }, [autoRefresh, selected])

  // Subscribe to SSE for the selected task
  useEffect(() => {
    es?.close()
    setEs(null)
    if (!selected) return
    const src = new EventSource(API(`/tasks/${selected.id}/events`))
    src.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data)
        if (data?.plan || data?.steps) {
          // snapshot case not used now
        }
      } catch {}
    }
    src.addEventListener('snapshot', (e:any) => {
      try { setSelected(JSON.parse(e.data)) } catch {}
    })
    src.addEventListener('update', (e:any) => {
      try {
        const ev = JSON.parse(e.data)
        if (!ev || !ev.event) return
        if (ev.event === 'task_status') {
          setSelected(prev => prev && prev.id===ev.task_id ? { ...prev, status: ev.payload?.status || prev.status } : prev)
          setTasks(prev => prev.map(t => t.id===ev.task_id ? { ...t, status: ev.payload?.status || t.status } : t))
        } else if (ev.event === 'plan') {
          setSelected(prev => prev && prev.id===ev.task_id ? { ...prev, plan: ev.payload } : prev)
        } else if (ev.event === 'step_status') {
          setSelected(prev => {
            if (!prev || prev.id!==ev.task_id) return prev
            const steps = prev.plan?.steps?.map((s:any) => s.id===ev.payload?.id ? { ...s, status: ev.payload?.status } : s)
            return { ...prev, plan: prev.plan ? { ...prev.plan, steps } : prev.plan }
          })
        } else if (ev.event === 'result') {
          setSelected(prev => {
            if (!prev || prev.id!==ev.task_id) return prev
            const results = [...(prev.results||[]), ev.payload]
            return { ...prev, results }
          })
        }
      } catch {}
    })
    setEs(src)
    return () => { src.close() }
  }, [selected?.id])

  async function createTask() {
    setBusy(true)
    try {
      const res = await fetch(API('/tasks'), {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query })
      })
      const data: Task = await res.json()
      setQuery('')
      setTasks(prev => [data, ...prev])
      setSelected(data)
    } finally { setBusy(false) }
  }

  async function startTask(id: string) {
    setBusy(true)
    try { await fetch(API(`/tasks/start/${id}`), { method: 'POST' }); await refresh() } finally { setBusy(false) }
  }

  async function planTask(id: string) {
    setBusy(true)
    try {
      const res = await fetch(API(`/tasks/plan/${id}`), { method: 'POST' })
      if (res.ok) await refresh()
    } finally { setBusy(false) }
  }

  async function executeTask(id: string) {
    setBusy(true)
    try { await fetch(API(`/tasks/execute/${id}`), { method: 'POST' }); await refresh() } finally { setBusy(false) }
  }

  async function fetchLLM() {
    try {
      const res = await fetch(API('/debug/llm'))
      if (res.ok) setLlmInfo(await res.json())
    } catch {}
  }

  const selectedId = selected?.id
  const statusBadge = (s?: string) => <span className={`badge ${s}`}>{s}</span>
  const sortedTasks = useMemo(() => {
    const clone = [...tasks]
    clone.sort((a,b) => {
      const ta = a.created_at ? Date.parse(a.created_at) : Number(a.id.split('-')[0])
      const tb = b.created_at ? Date.parse(b.created_at) : Number(b.id.split('-')[0])
      if (isNaN(tb - ta)) return (b.id > a.id ? 1 : -1)
      return (tb - ta)
    })
    return clone
  }, [tasks])

  return (
    <div className="app">
      <div className="card">
        <h2>New Task</h2>
        <div className="toolbar" style={{marginBottom:8}}>
          <input value={query} onChange={e=>setQuery(e.target.value)} placeholder="Enter query or URL" />
          <button className="btn primary lg" onClick={createTask} disabled={!query || busy}>Create</button>
        </div>
        <div className="toolbar small" style={{justifyContent:'space-between'}}>
          <div className="muted">API: {API_BASE}</div>
          <label><input type="checkbox" checked={autoRefresh} onChange={e=>setAutoRefresh(e.target.checked)} /> Auto refresh</label>
        </div>
        <div className="card" style={{marginTop:12}}>
          <div className="row" style={{marginBottom:8}}>
            <h2 style={{margin:0}}>LLM</h2>
            <button className="btn ghost sm" onClick={fetchLLM}>Refresh</button>
          </div>
          {!llmInfo ? <div className="muted small">Loading…</div> : (
            <div className="small">
              <div>Provider: <strong>{llmInfo.provider}</strong></div>
              <div>Model: <span className="muted">{llmInfo.model||'n/a'}</span></div>
              <div>Status: <span className={`badge ${llmInfo.ok? 'SUCCESS':'FAILED'}`}>{llmInfo.ok? 'OK':'ERROR'}</span> {llmInfo.error? <span className="muted">— {llmInfo.error}</span>:null}</div>
            </div>
          )}
        </div>

        <h2 style={{marginTop:16}}>Tasks</h2>
        <ul className="list">
          {sortedTasks.map(t => (
            <li key={t.id} className={`item ${selectedId===t.id ? 'selected':''}`}>
              <div className="row">
                <div className="name" style={{flex:1}}>
                  <div className="ellipsis title">{t.query || '(no query)'}</div>
                  <div className="muted">{t.id}</div>
                </div>
                <div className="toolbar">
                  {statusBadge(t.status)}
                  <button className="btn ghost sm" onClick={() => planTask(t.id)} disabled={busy}>Plan</button>
                  <button className="btn secondary md" onClick={() => executeTask(t.id)} disabled={busy || t.status==='RUNNING'}>Execute</button>
                  <button className="btn primary lg" onClick={() => startTask(t.id)} disabled={busy || t.status==='RUNNING'}>Start</button>
                  <button className="btn ghost sm" onClick={() => setSelected(t)}>Open</button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      </div>

      <div className="card">
        <h2>Task Detail</h2>
        {!selected ? <div className="muted">Select a task…</div> : (
          <div className="grid2">
            <div>
              <div className="small muted">ID</div>
              <div style={{marginBottom:8}}>{selected.id}</div>
              <div className="small muted">Status</div>
              <div style={{marginBottom:8}}>{statusBadge(selected.status)}</div>
              <div className="small muted">Query</div>
              <div style={{marginBottom:8, wordBreak:'break-word'}}>{selected.query}</div>
              <div className="toolbar" style={{gap:8}}>
                <button className="btn ghost sm" onClick={()=> planTask(selectedId!)} disabled={busy}>Plan</button>
                <button className="btn secondary md" onClick={()=> executeTask(selectedId!)} disabled={busy}>Execute</button>
                <button className="btn primary lg" onClick={()=> startTask(selectedId!)} disabled={busy}>Start</button>
              </div>
            </div>
            <div>
              <h2>Plan</h2>
              {!selected.plan ? <div className="muted small">Not planned yet.</div> : (
                <ul className="list">
                  {selected.plan.steps.map((s:any) => (
                    <li key={s.id} className="item">
                      <div className="row"><strong>{s.tool}</strong> {statusBadge(s.status)}</div>
                      <div className="muted small">{s.id} — {s.description}</div>
                      {s.inputs ? <pre>{JSON.stringify(s.inputs,null,2)}</pre> : null}
                    </li>
                  ))}
                </ul>
              )}
              <h2>Results</h2>
              {!selected.results?.length ? <div className="muted small">No results.</div> : (
                <ul className="list">
                  {selected.results.map((r:any,i:number) => (
                    <li key={i} className="item">
                      <div className="row"><strong>{r.step_id}</strong> <span className={`badge ${r.error? 'FAILED': (r.verified? 'SUCCESS':'PENDING')}`}>{r.error? 'ERROR': (r.verified? 'VERIFIED':'UNVERIFIED')}</span></div>
                      {r.logs ? <div className="muted small">{r.logs}</div> : null}
                      <pre>{typeof r.output === 'string' ? r.output : JSON.stringify(r.output,null,2)}</pre>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
