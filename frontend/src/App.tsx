import { useEffect, useState } from 'react'

type Task = {
  id: string
  query: string
  status: string
  plan?: { steps: Step[] }
  results?: Result[]
}
type Step = { id: string; description: string; tool: string; status: string }
type Result = { step_id: string; output?: any; logs?: string; verified: boolean; error?: string }

const API = (path: string) => `http://localhost:8080${path}`

export default function App() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<Task | null>(null)

  async function refresh() {
    const res = await fetch(API('/tasks'))
    const data = await res.json()
    setTasks(data)
    if (selected) {
      const sres = await fetch(API(`/tasks/${selected.id}`))
      setSelected(await sres.json())
    }
  }

  useEffect(() => { refresh(); const t = setInterval(refresh, 1500); return () => clearInterval(t) }, [])

  async function createTask() {
    const res = await fetch(API('/tasks'), {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query })
    })
    const data: Task = await res.json()
    setQuery('')
    setTasks(prev => [data, ...prev])
  }

  async function startTask(id: string) {
    await fetch(API(`/tasks/start/${id}`), { method: 'POST' })
    await refresh()
  }

  return (
    <div style={{display:'grid',gridTemplateColumns:'1fr 2fr',gap:16,padding:16,fontFamily:'Inter, system-ui, sans-serif'}}>
      <div>
        <h2>New Task</h2>
        <input value={query} onChange={e=>setQuery(e.target.value)} placeholder="Enter query or URL" style={{width:'100%',padding:8}} />
        <button onClick={createTask} style={{marginTop:8}}>Create</button>

        <h2 style={{marginTop:24}}>Tasks</h2>
        <ul style={{listStyle:'none',padding:0}}>
          {tasks.map(t => (
            <li key={t.id} style={{marginBottom:8, padding:8, border:'1px solid #ddd', borderRadius:6}}>
              <div style={{display:'flex',justifyContent:'space-between',alignItems:'center'}}>
                <div>
                  <strong>{t.id}</strong>
                  <div style={{fontSize:12,color:'#555'}}>{t.query}</div>
                </div>
                <div>
                  <span style={{marginRight:8}}>{t.status}</span>
                  <button onClick={() => startTask(t.id)} disabled={t.status==='RUNNING'}>Start</button>
                  <button onClick={() => setSelected(t)} style={{marginLeft:8}}>Open</button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      </div>
      <div>
        <h2>Task Detail</h2>
        {!selected ? <div>Select a task…</div> : (
          <div>
            <div><strong>ID:</strong> {selected.id}</div>
            <div><strong>Status:</strong> {selected.status}</div>
            <h3>Plan</h3>
            {!selected.plan ? <div>Not planned yet.</div> : (
              <ul>
                {selected.plan.steps.map(s => (
                  <li key={s.id}>[{s.status}] {s.tool} — {s.description}</li>
                ))}
              </ul>
            )}
            <h3>Results</h3>
            {!selected.results?.length ? <div>No results.</div> : (
              <ul>
                {selected.results.map((r,i) => (
                  <li key={i}>
                    step {r.step_id}: {r.verified ? 'verified' : 'not verified'} {r.error ? `— error: ${r.error}` : ''}
                    <pre style={{background:'#fafafa',padding:8,whiteSpace:'pre-wrap'}}>{typeof r.output === 'string' ? r.output : JSON.stringify(r.output,null,2)}</pre>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

