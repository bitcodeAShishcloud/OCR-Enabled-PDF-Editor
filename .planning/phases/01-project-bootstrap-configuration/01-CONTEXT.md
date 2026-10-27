# Phase 1: Project Bootstrap & Configuration - Context

**Gathered:** 2026-10-27
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers a runnable, CGO-free Go HTTP server with environment-driven configuration, an OCR engine abstraction, and backend selection. It does NOT include PDF I/O, OCR parsing, document store, or editing — those are later phases. The deliverable is a server that starts, reads config, selects an OCR backend, exposes a health endpoint, and shuts down gracefully.

</domain>

<decisions>
## Implementation Decisions

### HTTP Server
- **D-01:** Use Go standard library `net/http` with `http.ServeMux` (Go 1.22+ method patterns). No third-party router. Reversible.
- **D-02:** Listen address from `PORT` env var, default `8080`; bind `0.0.0.0` (all interfaces). Fail fast on invalid port. Reversible.
- **D-03:** Graceful shutdown on SIGINT/SIGTERM with a 10s shutdown timeout. Reversible.
- **D-04:** `GET /healthz` returns 200 OK JSON `{status:"ok"}`. Reversible.

### Configuration
- **D-05:** Parse env vars at startup into a typed `Config` struct; missing optional vars use documented defaults. Reversible.
- **D-06:** `TESSERACT_PATH` defaults to `C:\Program Files\Tesseract-OCR\tesseract.exe`; `FONT_PATH` defaults to `C:\Windows\Fonts\arial.ttf`; `OCR_BACKEND` defaults to `cli`. Reversible.
- **D-07:** Invalid `OCR_BACKEND` value fails startup with a clear error. Reversible.

### OCR Abstraction
- **D-08:** Define `ocr.Engine` interface in `internal/ocr/ocr.go`: `Recognize(imagePath string) ([]Word, error)`, with `Word{Text string, Confidence float64, X0,Y0,X1,Y1 int}`. Reversible.
- **D-09:** `internal/ocr/cli.go` default backend invokes tesseract binary; `internal/ocr/gosseract.go` behind `//go:build gosseract`. Reversible.
- **D-10:** Backend selection via factory `NewEngine(cfg) (Engine, error)` reading `OCR_BACKEND`. Reversible.

### Logging
- **D-11:** Use `log/slog` with JSON handler writing to stdout. Reversible.

### Module / Build
- **D-12:** Module path `ocrpdf` (already in go.mod). Default build CGO-free. Reversible.
- **D-13:** Add `internal/config` package? Decision: keep config in `cmd/server` main.go for now; extract if needed. Reversible.

### the agent's Discretion
- Exact file layout within `cmd/server` (main.go only).
- Error message wording.
- Whether to add a Makefile — deferred; use `go build`/`go test` directly.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project
- `.planning/PROJECT.md` — project context, constraints, key decisions
- `.planning/REQUIREMENTS.md` — CONF-01, CONF-02, OCR-01, OCR-06
- `.planning/ROADMAP.md` — Phase 1 goal and success criteria

### Research
- `.planning/research/SUMMARY.md` — stack and architecture recommendations
- `.planning/research/STACK.md` — Go toolchain, pdfcpu, gosseract, gg versions
- `.planning/research/ARCHITECTURE.md` — layered architecture, Engine interface

### Codebase
- `.planning/codebase/STACK.md` — technology stack (planned)
- `.planning/codebase/ARCHITECTURE.md` — planned architecture
- `.planning/codebase/STRUCTURE.md` — planned directory layout

No external specs beyond the above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `go.mod` / `go.sum` already declare all dependencies (pdfcpu, gosseract, gg, x/image, freetype). No source yet.

### Established Patterns
- Module path `ocrpdf`, Go 1.26.5. No existing code patterns (greenfield).

### Integration Points
- `cmd/server/main.go` is the entry point; will be created in this phase.
- `internal/ocr/ocr.go` defines the Engine interface consumed by all later phases.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches.

</specifics>

<deferred>
## Deferred Ideas

- Makefile / build scripts — use go tooling directly for v1.
- Structured request logging middleware — basic slog sufficient for v1.

None — discussion stayed within phase scope.

</deferred>

---
*Phase: 01-project-bootstrap-configuration*
*Context gathered: 2026-10-27*