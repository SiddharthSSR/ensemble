# Contributing to Ensemble

Thanks for your interest in contributing! This guide covers local setup, workflow, and CI/CD.

## Prerequisites
- Go 1.21+
- Node.js 20 + npm
- Google API Key (optional, for Gemini integration): `GOOGLE_API_KEY`

## Project Layout
- `backend/` — Go HTTP API, agents, orchestrator, tools
- `frontend/` — React + Vite app
- `.github/workflows/` — CI, Docker publish, GitHub Pages deploy

## Getting Started
1) Clone and install deps:
- Backend: `cd backend && go build ./...`
- Frontend: `cd frontend && npm install`

2) Run locally:
- Backend: `cd backend && go run ./cmd/server` (http://localhost:8080)
- Frontend: `cd frontend && npm run dev` (http://localhost:5173)

 

## Environment
- Backend:
  - `PORT` (default 8080)
  - `GOOGLE_API_KEY` (optional; Gemini integration stubbed for now)

## Git Workflow
- Create a branch from `main`:
  - Feature: `feat/<short-name>`
  - Fix: `fix/<short-name>`
  - Chore/Docs: `chore/<short-name>` or `docs/<short-name>`
- Commit style: Conventional Commits recommended (e.g., `feat(api): add tasks endpoint`)
- Push and open a Pull Request to `main`.

## Before Submitting a PR
- Backend: `go build ./... && go vet ./...`
- Frontend: `npm run build`
- Ensure CI passes on your branch.

## CI/CD
CI/CD is not configured currently. Use local builds or Docker Compose.

## Code Style
- Go: `go fmt`, `go vet`
- Frontend: default ESLint/Prettier not configured; keep code tidy and typed. We can add linters if needed.

## Security
- Do not commit secrets. Use GitHub Secrets for CI and GHCR.
- Tools are whitelisted; avoid adding unsafe execution paths.

## Questions
Open a GitHub Issue or start a discussion on the repo. PRs welcome!
