# Walking Skeleton - OCR PDF Text-Editing Backend (Go)

**Phase:** 1
**Generated:** 2026-10-27

## Capability Proven End-to-End

The service boots as a CGO-free HTTP process, reads its configuration from environment variables (with documented defaults), validates that Tesseract and the configured font exist, selects an OCR backend through a factory, serves a health endpoint, and shuts down gracefully on SIGINT/SIGTERM.

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| HTTP framework | Go stdlib `net/http` + `http.ServeMux` (Go 1.22+ method patterns) | Locked D-01; no third-party router needed for the planned routes. |
| Config location | Typed `Config` struct parsed in `cmd/server/main.go` | Locked D-13; keeps bootstrap in one place; extract later if needed. |
| OCR abstraction | `ocr.Engine` interface + `ocr.Config` + `NewEngine` factory | Locked D-08/D-10; decouples backend from API/edit layers; avoids import cycle. |
| Logging | `log/slog` JSON handler to stdout | Locked D-11; stdlib, structured. |
| CGO gating | `//go:build gosseract` on the gosseract backend file only | Locked D-12 / CONF-02; default build stays CGO-free. |
| Directory layout | `cmd/server`, `internal/ocr` | Standard Go layout; matches planned structure in `.planning/codebase/STRUCTURE.md`. |

## Stack Touched in Phase 1

- [x] Project scaffold (Go module already present; `cmd/server`, `internal/ocr` created)
- [x] Routing - `GET /healthz`
- [ ] Database - N/A (in-memory store deferred to Phase 4)
- [ ] UI - N/A (backend service)
- [x] Deployment - documented local run command (`go run ./cmd/server`)

## Out of Scope (Deferred to Later Slices)

- PDF I/O (extraction/rebuild) - Phase 2.
- TSV parsing and real OCR invocation - Phase 3.
- In-memory document store and upload/preview/cleanup API - Phase 4.
- Edit, rebuild, end-to-end verification - Phase 5.
- Makefile / build scripts - deferred; use `go build`/`go test` directly.
- Structured request-logging middleware - basic slog sufficient for v1.

## Subsequent Slice Plan

Each later phase adds one vertical slice on top of this skeleton without altering its architectural decisions:

- Phase 2: PDF I/O layer (embedded-raster extraction + dimension-preserving rebuild).
- Phase 3: OCR engine & TSV parsing (CLI + gosseract backends).
- Phase 4: In-memory document store & upload/preview/cleanup API.
- Phase 5: Edit, rebuild & end-to-end verification.