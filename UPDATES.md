# Project Updates

High-level change log for major refactors and features. Keep bullets concise.

## 2025-09-04

- LLM provider-agnostic layer:
  - Added `internal/providers/llm` with a minimal `Client` interface and `NewFromEnv()` factory.
  - Implemented HTTP clients for OpenAI (`openai.go`), Anthropic (`anthropic.go`), and Gemini (`gemini_http.go`), plus `mock.go` fallback.
- Planner/Verifier integration:
  - Switched `LLMPlanner` and `LLMVerifier` to use the generic `llm.Client`.
  - Improved planner parsing: strips code fences, extracts first JSON array; graceful fallback to trivial plan.
  - Verifier now parses JSON verdicts `{ "ok": bool, "reason": string }` when present.
- API wiring:
  - `internal/api/server.go` now selects the LLM client via `llm.NewFromEnv()` when `USE_LLM_PLANNER` / `USE_LLM_VERIFIER` are enabled.
- Configuration & docs:
  - Updated `backend/.env.example` with provider-agnostic settings: `LLM_PROVIDER`, `LLM_MODEL`, and provider-specific API keys.
  - Updated `README.md` to document multi-provider support and setup steps.
- Build verification:
  - Verified `go build ./...` succeeds after refactor.

