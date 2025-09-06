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
  - Adjusted LLM planner prompt to prefer 2–3 ordered steps using existing tools (echo, http_get) without cross-step data wiring.
- API wiring:
  - `internal/api/server.go` now selects the LLM client via `llm.NewFromEnv()` when `USE_LLM_PLANNER` / `USE_LLM_VERIFIER` are enabled.
- Configuration & docs:
  - Updated `backend/.env.example` with provider-agnostic settings: `LLM_PROVIDER`, `LLM_MODEL`, and provider-specific API keys.
  - Updated `README.md` to document multi-provider support and setup steps.
- Build verification:
  - Verified `go build ./...` succeeds after refactor.

## 2025-09-04 (later)

- Multi-step planning improvements:
  - Adjusted LLM planner prompt to encourage http_get → summarize flows and allow referencing prior outputs via `{{step:ID.output}}`.
- Summarize tool:
  - Added `tools.SummarizeTool` backed by the configured LLM (`llm.Client.GenerateText`).
  - Registered in API server so plans can execute summarize steps.
- Orchestrator input resolution:
  - Added simple input substitution to map `{{step:ID.output}}` to the stringified output of prior steps.
  - Applies to both plan+run and execute-existing-plan paths.
 - Docs updated to mention `summarize` tool and output-referencing template.

## 2025-09-04 (tools)

- New tools:
  - `http_post_json`: POST JSON to APIs with optional headers; returns response body and status in logs.
  - `html_to_text`: Convert HTML to readable text using stdlib HTML parser; use before summarization.
  - `llm_answer`: Ask the configured LLM to answer a question directly.
- Planner prompt:
  - Now recommends URL flow: http_get → html_to_text → summarize; and uses llm_answer for direct questions.
- Docs:
  - Added examples for the new tools in README.

## 2025-09-05

- UI polishing:
  - Dark theme, status badges, button priority (primary/secondary/ghost; lg/md/sm), ellipsis for long task titles, selected-task highlight.
  - LLM debug panel shows provider/model/health; added Plan/Execute controls and auto-refresh toggle.
- Planner defaults & precedence:
  - Non-URL queries now default to `llm_answer` (instead of `echo`).
  - URL queries default to a 3-step chain: `http_get` → `html_to_text` → `summarize`.
- LLM robustness:
  - Added `LLM_HTTP_TIMEOUT_MS`, simple retries/backoff, and provider base URL overrides (`OPENAI_API_BASE`, `ANTHROPIC_API_URL`, `GEMINI_API_URL`).
- Debugging:
  - `/debug/llm` endpoint to diagnose provider/model connectivity.

- Live updates (SSE):
  - Added in-memory event hub and `/tasks/{id}/events` endpoint for Server-Sent Events.
  - Streams: task status changes, plan snapshot, step status updates, and results.

## 2025-09-06–07 (Unified tools, files, performance)

- Unified planning & execution:
  - Added `call_tool` delegator so the planner can emit a single tool that calls any registered tool. Enable via `USE_UNIFIED_TOOL=1`.
  - Deep input substitution now replaces `{{step:ID.output}}` inside nested maps/arrays.
- New tools:
  - `file_extract` (generic attachments → text; PDF/HTML/text/CSV/JSON/YAML autodetect), `extract_links`, `csv_parse`, `regex_extract`, `json_pretty`, `summarize_chunked` (map-reduce).
- Files/attachments:
  - Frontend now accepts any file; sends `context.attachments=[{data_base64, filename, content_type}]` (keeps `pdf_data_base64` for back-compat).
  - Planner: CSV → `file_extract` → `csv_parse` → summarize/answer; JSON → `file_extract` → `json_pretty` → summarize/answer.
- Performance & stability:
  - Output caps with env vars: `FILE_MAX_BYTES`, `PDF_MAX_TEXT_BYTES`, `HTML_TO_TEXT_MAX_BYTES`, `HTTP_GET_MAX_BYTES`.
  - SSE token coalescing at 100ms on backend; UI also throttles streaming renders at 100ms.
  - UI clamps large outputs/streams with Expand/Collapse to prevent blank screens and scroll thrash.
  - Added `summarize_chunked` and planner preference for chunked summary on large texts.
