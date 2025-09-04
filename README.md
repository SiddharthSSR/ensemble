# Agent Orchestrator (Go + React)

## Overview
Planner → Executor(s) → Verifier pipeline with a Go backend and React frontend. Gemini integration is stubbed for now; a mock planner and simple verifier are included. Tools are pluggable (echo, http_get provided).

## Project Structure
- backend/
  - cmd/server: entrypoint
  - internal/{api,agents,models,orchestrator,providers/gemini,tools}
- frontend/
  - Vite + React + TS app

## Run Backend
```
cd backend
GOOGLE_API_KEY=your_key # optional for future Gemini integration
go run ./cmd/server
```
Server listens on :8080.

## Run Frontend
```
cd frontend
npm install
npm run dev
```
App runs on http://localhost:5173 and talks to backend at http://localhost:8080.

## API
- POST /tasks { query, context? } → Task
- POST /tasks/start/{id} → 202 Accepted, starts orchestration
- GET /tasks → list
- GET /tasks/{id} → details (plan, steps, results)

## Notes
- Planner: rule-based mock. Replace with Gemini client in `internal/providers/gemini` and a real planner in `internal/agents`.
- Safety: tools are whitelisted. No arbitrary code execution.
- Persistence: in-memory for MVP. Swap with a store if needed.

## Notes
- Planner: rule-based mock. Replace with Gemini client in `internal/providers/gemini` and a real planner in `internal/agents`.
- Safety: tools are whitelisted. No arbitrary code execution.
- Persistence: in-memory for MVP. Swap with a store if needed.

## Docker / Compose
- Build and run both services locally:
  - `docker compose up --build`
  - Backend: http://localhost:8080
  - Frontend: http://localhost:8081
- Images:
  - Backend: builds from `backend/Dockerfile`
  - Frontend: builds static site with Node then serves via Nginx
