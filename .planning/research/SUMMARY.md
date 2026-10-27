# Project Research Summary

**Project:** OCR PDF Text-Editing Backend (Go)
**Domain:** Go HTTP backend for scanned-PDF OCR text replacement
**Module:** `ocrpdf`
**Researched:** 2026-10-27
**Confidence:** HIGH

## Executive Summary

This is a single-process, session-scoped Go HTTP backend that replaces text inside scanned (raster) PDFs. A client uploads a scanned PDF; the service extracts the embedded raster image from each page, runs Tesseract OCR to obtain word-level bounding boxes, and exposes a preview. The user then requests an in-place replacement of a detected word; the service erases the old glyphs using a sampled background color and renders the new text fitted to the original bbox, then rebuilds the output PDF from the edited page images. The architecture is deliberately small and dependency-light: standard library `net/http` only, pdfcpu for PDF I/O, Tesseract for OCR, and fogleman/gg for raster rendering.

Research recommends a layered, interface-based design: `cmd/server` bootstraps config and selects an OCR backend; `internal/api` owns the HTTP handlers and an in-memory, mutex-protected document store; `internal/pdfio`, `internal/ocr`, and `internal/edit` are independent leaf packages (no cross-imports) wrapping pdfcpu, Tesseract, and gg respectively. The OCR backend is hidden behind an `ocr.Engine` interface with a default pure-Go-friendly CLI implementation and an optional CGO gosseract implementation gated behind `//go:build gosseract`. This keeps the default build CGO-free (critical on Windows where Tesseract dev headers are absent) while leaving a seam for the faster C++ backend.

The dominant risks are not framework choices but correctness hazards in the raster pipeline: pdfcpu's extraction returns only an error (so "largest image per page" must be derived by enumerating outputs), `ImportImagesFile` appends to an existing output file (must truncate first), the default `Import` config stretches images to A4 (must set `PageDim` from the source), Tesseract TSV parsing is fragile (header, empty lines, `-1` confidence, trailing empty fields), and coordinate/baseline alignment must stay in image pixel space. Concurrency hazards (holding the mutex across I/O, per-document races, Windows file locking) and unbounded memory on large PDFs are the operational risks. All are preventable with the documented mitigations and must be addressed in the first implementation phases rather than retrofitted.

## Key Findings

### Recommended Stack

Pure-Go-first stack on Go 1.26.5 with a single external binary dependency (Tesseract 5.5.0). PDF I/O uses pdfcpu v0.16.3 (pure Go, Apache-2.0) for lossless embedded-raster extraction and rebuild. Raster rendering uses fogleman/gg v1.3.0 + golang.org/x/image v0.47.0 + golang/freetype (indirect). OCR is Tesseract via `os/exec` by default, with optional gosseract/v2 v2.4.1 behind a build tag. HTTP/JSON/CSV/logging/multipart all come from the standard library; state is an in-memory `sync.Mutex`-protected map (no DB).

**Core technologies:**
- Go 1.26.5 toolchain — stdlib `net/http` enhanced routing (1.22+) and experimental `encoding/json/v2`; module `ocrpdf`
- pdfcpu v0.16.3 — `ExtractImagesFile` / `ImportImagesFile` / `DefaultImport` config; pure Go, no CGO; pin (pre-1.0 API drift risk)
- Tesseract 5.5.0.20241111 (external binary at `C:\Program Files\Tesseract-OCR\tesseract.exe`, override `TESSERACT_PATH`) — CLI backend, TSV with 12 columns incl. bbox/conf; `gosseract/v2 v2.4.1` optional behind `//go:build gosseract`
- fogleman/gg v1.3.0 + golang.org/x/image v0.47.0 + freetype — erase (fill padded bbox) + render replacement text with font-fit and color sampling; keep behind `internal/edit` (gg unmaintained since 2019)
- stdlib only for HTTP/JSON/CSV/logging/multipart — no web framework; `encoding/csv` for robust TSV parsing

### Expected Features

Five REST endpoints form the table-stakes surface: upload+OCR all pages, page PNG preview, edit by oldText/wordIds, rebuild/download result PDF, and cleanup. Differentiators are the dual OCR backend, background-sampling erase, font-fit render, text-color sampling, auto-fit with overrides, lossless extraction, and synthetic E2E verification. Explicit anti-features (no auth, no DB, no vector rasterization, no multi-tenant, no streaming, no batch edit, no cloud storage, no observability stack, no language UI, no horizontal scaling) keep scope tight and single-process.

**Must have (table stakes):**
- Upload scanned PDF → doc id + per-page word-level OCR (`POST /api/documents`)
- Word-level OCR with pixel bboxes (x0,y0,x1,y1) via Tesseract TSV level-5 rows
- Per-page preview PNG (`GET .../pages/{n}`)
- Replace detected word by oldText or wordIds (`POST .../edits`) with erase + render
- Rebuild/download final PDF (`GET .../result.pdf`)
- Cleanup document + artifacts (`DELETE ...`)
- In-memory mutex-protected document store (process-local, lost on restart — accepted)
- Configurable env vars (`TESSERACT_PATH`, `TESSDATA_PREFIX`, `OCR_BACKEND`, `FONT_PATH`)

**Should have (competitive):**
- Dual OCR backend behind common `ocr.Engine` interface (CLI default, gosseract opt-in)
- Background sampling erase (median ring sampling, padded rect fill)
- Font-fit render (binary-search font size to bbox width, baseline-aligned, color sampling)
- Lossless embedded-image extraction (largest raster per page)
- Synthetic end-to-end verification (draw name → import → OCR → edit → re-OCR)
- Unit tests for TSV parsing and font-fit

**Defer (v2+):**
- gosseract backend (ship CLI default first; add CGO variant when headers available)
- Text color sampling polish (start with black fallback, then sample darkest glyphs)
- Optional font/size/color overrides (after auto-fit baseline works)
- Synthetic E2E harness (build alongside MVP to validate core value)
- Bounded OCR worker pool, TTL eviction, result-PDF caching (recommended beyond minimal plan)

### Architecture Approach

Layered HTTP service with strict one-way dependency direction: `cmd/server -> internal/api -> internal/{pdfio,ocr,edit} -> external libs`. No package imports `internal/api`; the three leaf packages do not import each other. `internal/api` owns the `Store` (mutex + map), route registration, and orchestration. Artifacts live on disk under a per-document temp dir (`uploads/`, `pages/`, `edits/`, `result/`). IDs are server-generated (crypto/rand), never user-supplied, to prevent path traversal.

**Major components:**
1. `cmd/server/main.go` — env config, OCR backend selection, router wiring, `http.Server` startup/graceful shutdown
2. `internal/api` (`handlers.go`) — HTTP handlers, validation, orchestration, in-memory `Store` with mutex
3. `internal/pdfio` (`extract.go`, `build.go`) — PDF↔raster bridge; `ExtractImagesFile` + `ImportImagesFile` wrapped and isolated
4. `internal/ocr` (`ocr.go`, `cli.go`, `gosseract.go`) — `Engine` interface; CLI default; gosseract behind `//go:build gosseract`
5. `internal/edit` (`erase.go`, `render.go`) — in-place replacement: erase old glyphs, render new text (gg/x/image/font)

Patterns to follow: Engine interface (Strategy), constructor injection with no globals, per-request context + timeouts (with `exec.CommandContext`), path safety (generated IDs + containment checks), and atomic-ish file writes (temp + rename).

### Critical Pitfalls

Top pitfalls from research (see PITFALLS.md for full list and phase-specific warnings):

1. **pdfcpu `ExtractImagesFile` returns only an error, not filenames** — a page can yield multiple XObjects (masks/thumbnails); "largest image per page" must be derived by enumerating `outDir`, decoding dimensions, and selecting max pixel area. Clean dir per doc; treat `*pdfcpu.UnsupportedResourceError` as failure.
2. **`ImportImagesFile` appends to an existing output file** — always `os.Remove(outFile)` first or use a fresh temp path; validate page count equals image count.
3. **Default `Import` config defaults to A4 and full-page stretch** — set `PageDim` from the source page/image dimensions; preserve per-page sizes; verify aspect ratio.
4. **Tesseract TSV parsing fragility** — use `encoding/csv` (tab delimiter, `FieldsPerRecord=-1`); skip header/blank lines; accept only `level==5` with `conf>=0`; `ParseFloat` for confidence; handle variable field counts.
5. **Image coordinate-system mismatch** — stay in image pixel space (top-left origin); map Tesseract bbox directly; never convert to PDF points inside `internal/edit`; handle `/Rotate` explicitly.
6. **Font baseline alignment** — compute baseline from font metrics (ascent/descent/height), not bbox bottom; binary-search font size on width AND enforce height fits bbox; clamp.
7. **Background erase on non-uniform scans** — median (not mean) per channel over a wider ring; exclude dark pixels; pad bbox 2–4 px; make sampling configurable.
8. **CGO build-tag leakage** — guard every gosseract file (incl. tests) with `//go:build gosseract`; default build `CGO_ENABLED=0`; gosseract needs MinGW-built headers on Windows (MSVC installer is ABI-incompatible).
9. **In-memory store leaks and races** — per-document mutex with consistent lock ordering (store → doc, never reverse); never copy mutexes; `DELETE` removes map entry under lock then files outside; run `go test -race`.
10. **Temp file cleanup and Windows file locking** — one `MkdirTemp` per doc; close all handles before removal; `defer os.RemoveAll`; UUID IDs with containment checks; `MaxBytesReader` + PDF magic-byte validation.

## Implications for Roadmap

Based on combined research, suggested phase structure (dependency-driven, smallest vertical slice first):

### Phase 1: Project Bootstrap & Configuration
**Rationale:** Establish the build/runtime foundation that every later phase depends on; default build must stay CGO-free from day one.
**Delivers:** `cmd/server/main.go` skeleton, env config (`TESSERACT_PATH`, `TESSDATA_PREFIX`, `OCR_BACKEND`, `FONT_PATH`, `PORT`), OCR backend selection, router wiring, graceful shutdown; `internal/ocr` package with `Engine` interface and CLI implementation; startup validation that Tesseract binary and font exist.
**Addresses:** Configurable env vars (table stakes); dual OCR backend scaffold (differentiator).
**Avoids:** Pitfall 8 (CGO build-tag leakage) — gosseract file guarded from the start; Pitfall "Tesseract not on PATH" / "Font file missing" — fail fast with clear messages.

### Phase 2: PDF I/O Layer (`internal/pdfio`)
**Rationale:** Extraction and rebuild are the two highest-risk pdfcpu touch points; isolate and verify them before wiring into HTTP.
**Delivers:** `extract.go` (call `ExtractImagesFile`, enumerate outputs, pick largest by pixel area per page, clean dir, handle `UnsupportedResourceError`) and `build.go` (truncate output, set `PageDim` from source, preserve per-page sizes, validate page count).
**Addresses:** Lossless embedded-image extraction (differentiator); upload/rebuild prerequisites (table stakes).
**Avoids:** Pitfalls 1, 2, 3, 10 (multiple images per page; append-to-existing; A4 stretch; temp/locking); soft-mask selection heuristic.

### Phase 3: OCR Engine & TSV Parsing
**Rationale:** Word-level bboxes with confidence are the data contract for all downstream editing; the TSV parser is fragile and needs deterministic tests.
**Delivers:** CLI backend (`exec.CommandContext`, parse 12-column TSV via `encoding/csv`, filter level-5/conf>=0, `ParseFloat`), `Word` type, unit tests with golden TSV (header, empty text, `-1` rows, blank lines). Optionally the gosseract backend behind `//go:build gosseract` using `GetBoundingBoxesVerbose()`.
**Addresses:** Word-level OCR with bboxes (table stakes); dual OCR backend (differentiator).
**Avoids:** Pitfalls 4, 8, "gosseract API name mismatch"; Pitfall 12 (concurrency cap via semaphore recommended).

### Phase 4: In-Memory Document Store & Upload Endpoint
**Rationale:** The store is the single source of truth and the concurrency seam; get locking and lifecycle right before edits.
**Delivers:** `Store` with `sync.RWMutex` (map-level) + per-document mutex, server-generated UUID IDs, `POST /api/documents` (multipart upload with `MaxBytesReader`, magic-byte validation, size/page caps, extract + OCR all pages, store record), `GET .../pages/{n}` (serve scoped PNG), `DELETE ...` (remove record + `os.RemoveAll` dir).
**Addresses:** Upload+OCR, preview PNG, cleanup, in-memory store (table stakes).
**Avoids:** Pitfalls 9, 10, 11, 12 (leaks/races, Windows locking/traversal, large-PDF bounds); Anti-Patterns 2, 3, 5, 8.

### Phase 5: Edit Pipeline (Erase + Render)
**Rationale:** The core value proposition; depends on OCR bboxes and original page images; highest heuristic risk.
**Delivers:** `internal/edit/erase.go` (median ring sampling, padded fill, clamp to bounds, configurable padding/sampling, optional no-erase/explicit bg), `render.go` (TTF load via `FONT_PATH`, metrics-based baseline, binary-search font size on width+height, text color sampling with black fallback, explicit overrides), `POST /api/documents/{id}/edits` (resolve by oldText/wordIds, write edited image atomically to `edits/`, mark page edited, return preview).
**Addresses:** Replace by oldText/wordIds (table stakes); background erase, font-fit, color sampling, overrides (differentiators).
**Avoids:** Pitfalls 5, 6, 7 (coordinate mismatch, baseline/height overflow, non-uniform backgrounds); Anti-Pattern 6 (mutating originals — keep `pages/` immutable, write to `edits/`).

### Phase 6: Result Build & Download
**Rationale:** Deliverable artifact; depends on edited/original image selection and pdfio rebuild correctness.
**Delivers:** `GET /api/documents/{id}/result.pdf` (per-page edited-if-present-else-original, rebuild via `ImportImagesFile`, cache result per doc invalidated on edit, stream attachment with `Content-Disposition`).
**Addresses:** Rebuild/download PDF (table stakes).
**Avoids:** Pitfalls 2, 3, "result PDF page size vs edited image size"; stale-cache pitfall (invalidate on edit).

### Phase 7: Verification & Hardening
**Rationale:** Lock in correctness with deterministic and end-to-end tests before declaring MVP done.
**Delivers:** Unit tests (TSV parsing, font-fit), race-detector CI (`go test -race ./...`), gosseract compile-only CI job (`go build -tags gosseract`), synthetic E2E harness (draw name onto image via gg → pdfcpu import → OCR → edit → re-OCR verifies new text), Windows-specific run, bounded OCR semaphore, TTL/eviction or max-doc cap.
**Addresses:** Synthetic E2E verification (differentiator); unit tests (differentiator).
**Avoids:** Pitfalls 8, 9, 12; "Race detector not run".

### Phase Ordering Rationale

- Dependencies flow strictly: bootstrap → pdfio → OCR → store/upload → edit → result → verification. Each phase's outputs are the next phase's inputs (OCR bboxes feed edits; original images feed both edits and rebuild).
- pdfio and OCR are isolated into their own phases because they are the two highest-risk external-lib touch points and must be verified (with tests) before HTTP wiring.
- The store/upload phase precedes edits because edits need the document record, word lists, and preview serving; per-document locking must exist before concurrent edits are possible.
- Result build is separated from edits so the cache-invalidation and image-selection logic is testable on its own.
- Verification is last so the synthetic E2E harness exercises the full pipeline; however, unit tests for TSV parsing and font-fit should be co-located with their phases (Phases 3 and 5), not deferred.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (pdfio):** pdfcpu v0.16.3 exact signatures (`ExtractImagesFile`, `ImportImagesFile`, `PageDimsFile`, `ValidateImportImagesOutput`) must be verified against installed version; `Import` struct fields (`PageDim`, `Pos`, `Scale`, `DPI`) need confirmation. Needs `/gsd-plan-phase --research-phase`.
- **Phase 5 (edit):** gg metric APIs (`MeasureString`, font ascent/descent) and color-sampling heuristic need empirical tuning; baseline math must be validated. Needs `/gsd-plan-phase --research-phase`.
- **Phase 7 (gosseract variant):** Windows MinGW header provisioning and `GetBoundingBoxesVerbose` field names need validation on a machine with headers (none available locally). Needs `/gsd-plan-phase --research-phase` (or defer until headers available).

Phases with standard, well-documented patterns (skip research-phase):
- **Phase 1 (bootstrap):** standard Go `main.go`/`http.Server`/env config.
- **Phase 3 (OCR CLI):** `os/exec` + `encoding/csv` are standard library; TSV format is documented.
- **Phase 4 (store/upload):** `sync.Mutex` map + multipart handling are standard Go patterns.
- **Phase 6 (result build):** streaming `http.ServeFile`/`io.Copy` is standard.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Versions verified against pkg.go.dev and installed toolchain (Go 1.26.5, Tesseract 5.5.0); go.mod present |
| Features | HIGH | Derived from PROJECT.md, codebase map, and the implementation plan; REST table specified |
| Architecture | HIGH | Layered pattern with interface seam is well-established; component boundaries specified at file level |
| Pitfalls | MEDIUM-HIGH | Verified against pdfcpu v0.16.3 pkg.go.dev, Tesseract TSV spec, gosseract Windows notes; concurrency/memory guidance from stdlib docs |

**Overall confidence:** HIGH

### Gaps to Address

- **pdfcpu API exactness:** `ExtractImagesFile`/`ImportImagesFile` signatures and the `Import`/`ImportParam` field names must be re-verified at implementation time against v0.16.3; wrap in `internal/pdfio` to isolate drift. (How to handle: integration tests on fixture PDFs in Phase 2.)
- **gosseract availability:** No Tesseract dev headers on the dev machine, so the CGO backend cannot be built/tested locally. (How to handle: keep behind build tag; ship CLI default; add CI job on Linux; defer gosseract to a later phase or a machine with headers.)
- **gg maintenance:** fogleman/gg is unmaintained (2019) but stable; keep usage isolated in `internal/edit` so a future swap is localized.
- **Font availability on non-Windows:** Default `C:\Windows\Fonts\arial.ttf` is Windows-only. (How to handle: `FONT_PATH` override + fail-fast startup message + documented fallback.)
- **No persistence:** In-memory store is lost on restart (accepted anti-feature); if future milestones need multi-instance, introduce a `Store` interface now (already planned) to swap backends without rewriting handlers.
- **Large-PDF limits:** Sync OCR with no background jobs; document the limitation and enforce size/page caps + timeouts from Phase 4.
- **Color sampling reliability:** Heuristic may fail on complex backgrounds; keep explicit `color` override escape hatch and black fallback.

## Sources

### Primary (HIGH confidence)
- pkg.go.dev: pdfcpu v0.16.3, gosseract v2.4.1, gg v1.3.0, golang.org/x/image v0.47.0, golang/freetype (verified 2026-10-27)
- Installed toolchain: `go version go1.26.5 windows/amd64`; Tesseract 5.5.0.20241111 at `C:\Program Files\Tesseract-OCR\tesseract.exe`
- Project: `.planning/PROJECT.md`, `.planning/codebase/{STACK,ARCHITECTURE,STRUCTURE,INTEGRATIONS,CONCERNS}.md`

### Secondary (MEDIUM confidence)
- pdfcpu v0.16.3 API docs (ExtractImagesFile, ImportImagesFile, DefaultImportConfig, PageDimsFile) — HIGH for API existence, MEDIUM for field names
- pdfcpu Import struct source (v0.16.3 importImage.go)
- Tesseract TSV format docs (tomrochette.com, tesseract-ocr.github.io)
- fogleman/gg API (github.com/fogleman/gg)
- gosseract Windows CGO notes (github.com/otiai10/gosseract/knowledge/windows-support.md)
- Go standard library: `sync`, `os`, `net/http`, race detector docs

### Tertiary (LOW confidence)
- Tesseract coordinate system origin conventions (image top-left vs PDF bottom-left) — MEDIUM, needs round-trip test
- Color-sampling / median-ring erase heuristic — inferred, needs empirical validation

---
*Research completed: 2026-10-27*
*Ready for roadmap: yes*
