# Repository Guidelines

## Project Structure & Module Organization
- `backend/`: Go 1.24 service. Entry at `backend/cmd/server/main.go`; internal packages live under `backend/internal/` (API routes, orchestrator, agents, tools, providers).
- `frontend/`: React + TypeScript app (Vite). Source in `frontend/src/`, HTML shell in `frontend/index.html`.
- Docs: `README.md`, `UPDATES.md`, and this guide. Example env in `backend/.env.example`.

## Build, Test, and Development Commands
- Backend
  - Init env: `cp backend/.env.example backend/.env` and set `LLM_PROVIDER` and the matching API key.
  - Run dev: `cd backend && go run ./cmd/server` (listens on `:8080`; override with `PORT`).
  - Health checks: `curl :8080/health` and `curl :8080/debug/llm`.
  - Lint/format: `cd backend && go vet ./... && go fmt ./...`.
- Frontend
  - Install: `cd frontend && npm install`.
  - Dev server: `npm run dev` (expects backend at `http://localhost:8080`).
  - Build/preview: `npm run build && npm run preview`.

## Coding Style & Naming Conventions
- Go: use `go fmt`. Packages are lower_snake, exported names in PascalCase. Prefer table-driven logic, explicit errors (`if err != nil`), and context-aware calls.
- TypeScript/React: 2-space indent, strict TS (`tsconfig.json`). Components in PascalCase (e.g., `App.tsx`), functions/vars in `camelCase`. Keep UI state local; extract reusable UI to `src/components/` when it grows.

## Testing Guidelines
- Currently no test suite in repo. For backend changes, add Go `testing` package tests (`*_test.go`) and run `go test ./...`.
- For frontend logic-heavy additions, consider introducing Vitest + React Testing Library in `frontend/` and co-locate tests as `Component.test.tsx`.

## Commit & Pull Request Guidelines
- Commits follow conventional style: `feat(scope): ...`, `fix(scope): ...`, `docs: ...`, `chore: ...` (see `git log`). Keep commits focused.
- PRs should include: clear description, linked issues, screenshots/GIFs for UI, API endpoints touched, and any `.env` changes. Add manual test notes (steps + expected behavior).
- Ensure CI-friendly steps: repo builds cleanly and formatting passes.

## Security & Configuration Tips
- Never commit secrets. Use `backend/.env` (ignored) and update `backend/.env.example` when adding new vars.
- Feature toggles: `USE_LLM_PLANNER`, `USE_LLM_VERIFIER`. Providers via `LLM_PROVIDER` (`openai|anthropic|gemini`) and optional `LLM_MODEL`.
