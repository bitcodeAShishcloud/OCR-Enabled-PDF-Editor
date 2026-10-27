<!-- refreshed: 2026-10-27 -->
# Architecture

**Analysis Date:** 2026-10-27

## Current State

As of 2026-10-27 the repository contains only `go.mod`, `go.sum`, and the design plan at `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`. No Go source files exist yet, so there is no implemented runtime architecture to describe. The sections below capture the **intended/target architecture** defined by that plan, which is the blueprint implementation should follow. All file paths are therefore planned (not yet present) unless noted.

## System Overview

```text
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Client (browser)                   │
└────────────────────────────┬────────────────────────────────┘
                             │ REST (JSON + PNG + PDF)
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                   cmd/server/main.go                         │
│   env config (TESSERACT_PATH, TESSDATA_PREFIX, OCR_BACKEND,  │
│   FONT_PATH) + net/http server startup                       │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│              internal/api/handlers.go                        │
│   REST handlers + in-memory document store (mutex map)       │
├──────────────────┬──────────────────┬───────────────────────┤
│   pdfio package  │   ocr package    │   edit package        │
│ internal/pdfio   │ internal/ocr     │ internal/edit         │
│ extract / build  │ Engine interface │ erase / render        │
└────────┬─────────┴────────┬─────────┴──────────┬────────────┘
         │                  │                     │
         ▼                  ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│  External: pdfcpu (extract/build PDF)  │  Tesseract (OCR)   │
│  github.com/pdfcpu/pdfcpu             │  CLI or gosseract   │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│  Raster store (local filesystem)                            │
│  uploaded PDFs, per-page PNGs, edited images, result.pdf     │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Server bootstrap | Read env config, wire OCR backend selection, start `net/http` server | `cmd/server/main.go` |
| REST handlers + doc store | Upload/OCR orchestration, page image serving, edit application, PDF rebuild, cleanup; thread-safe in-memory document map | `internal/api/handlers.go` |
| PDF I/O | Extract largest embedded raster image per scanned page (`ExtractImagesFile`); rebuild a PDF from edited images (`ImportImagesFile`) | `internal/pdfio/extract.go`, `internal/pdfio/build.go` |
| OCR abstraction | `Engine` interface: `Recognize(imagePath) ([]Word, error)`; `Word{text, confidence, x0, y0, x1, y1}` | `internal/ocr/ocr.go` |
| OCR CLI backend (default) | Invoke `tesseract <img> stdout tsv`, parse 12-column TSV level-5 rows | `internal/ocr/cli.go` |
| OCR gosseract backend (optional) | `gosseract/v2` `GetTextBoxes`, compiled only with `gosseract` build tag | `internal/ocr/gosseract.go` |
| Erase | Sample background color from the pixel ring around a bbox, fill padded rect | `internal/edit/erase.go` |
| Render | `gg`: auto-fit font size to bbox, sample text color from old glyphs, draw replacement text baseline-aligned | `internal/edit/render.go` |

## Pattern Overview

**Overall:** Layered HTTP service with an interface-based OCR backend and an in-memory document store. Each edited page is flattened into a raster image; the output PDF is rebuilt from those images.

**Key Characteristics:**
- Standard library only for HTTP (`net/http`); no third-party web framework declared in `go.mod`
- OCR behind a single `ocr.Engine` interface with two implementations selected at startup via `OCR_BACKEND` and a build tag
- PDF processing is lossless extraction of embedded scanned images (vector page rasterization is out of scope)
- Stateless-per-request handlers backed by a mutex-protected in-memory document map (no database)
- Heavy use of external binaries/libraries: Tesseract (CLI or CGO), pdfcpu, fogleman/gg

## Layers

**Entry / Bootstrap Layer:**
- Purpose: Configuration and process startup
- Location: `cmd/server/`
- Contains: `main.go`
- Depends on: `internal/api`, `internal/ocr`, env vars
- Used by: OS process / container entrypoint

**API Layer:**
- Purpose: HTTP routing, request validation, orchestration of pdfio/ocr/edit, document lifecycle
- Location: `internal/api/`
- Contains: `handlers.go`
- Depends on: `internal/pdfio`, `internal/ocr`, `internal/edit`
- Used by: `cmd/server/main.go`

**PDF I/O Layer:**
- Purpose: Bridge between PDF files and per-page raster images (input extraction and output rebuild)
- Location: `internal/pdfio/`
- Contains: `extract.go`, `build.go`
- Depends on: `github.com/pdfcpu/pdfcpu`
- Used by: `internal/api`

**OCR Layer:**
- Purpose: Recognize words + bounding boxes from a page image
- Location: `internal/ocr/`
- Contains: `ocr.go` (interface + `Word`), `cli.go` (default), `gosseract.go` (optional CGO)
- Depends on: Tesseract binary or `gosseract/v2`
- Used by: `internal/api`

**Edit Layer:**
- Purpose: In-place text replacement on a raster page (erase old glyphs, render new text)
- Location: `internal/edit/`
- Contains: `erase.go`, `render.go`
- Depends on: `github.com/fogleman/gg`, `golang.org/x/image`, font files
- Used by: `internal/api`

## Data Flow

### Primary Request Path (upload + OCR)

1. Client uploads PDF via `POST /api/documents` (entry: `internal/api/handlers.go`)
2. `internal/pdfio/extract.go` calls pdfcpu `api.ExtractImagesFile` to pull the largest embedded raster image per page into local files
3. `internal/ocr` `Engine.Recognize` runs over each page image (CLI: `tesseract img stdout tsv`; gosseract: `GetTextBoxes`)
4. Document record (id, page images, word lists) stored in the mutex-protected in-memory map in `internal/api/handlers.go`
5. JSON response `{docId, pages:[{page, words:[{text, conf, bbox}]}]}` returned to client

### Edit Path

1. Client calls `POST /api/documents/{id}/edits` with `{page, oldText or wordIds, newText, font?, size?, color?}` (`internal/api/handlers.go`)
2. `internal/edit/erase.go` samples background color from the pixel ring around the target bbox and fills the padded rect
3. `internal/edit/render.go` loads the TTF (default `C:\Windows\Fonts\arial.ttf`), binary-searches font size to fit the old bbox width, samples text color from the old glyphs' darkest pixels, draws the new text baseline-aligned to the bbox bottom
4. Edited preview image served back to client

### Result Build Path

1. Client calls `GET /api/documents/{id}/result.pdf` (`internal/api/handlers.go`)
2. `internal/pdfio/build.go` calls pdfcpu `api.ImportImagesFile` (one image per page) to rebuild the PDF from current edited images
3. Result PDF streamed to client

### Cleanup Path

1. Client calls `DELETE /api/documents/{id}` (`internal/api/handlers.go`)
2. In-memory record removed and associated local files (uploaded PDF, page images, edited images, result) deleted from disk

### State Management

- No database; document state lives in an in-memory, mutex-protected map inside `internal/api/handlers.go`
- Page images and edited images are materialized as files on the local filesystem; the store holds references/paths

## Key Abstractions

**`ocr.Engine`:**
- Purpose: Decouple OCR implementation from the API and edit layers
- Examples: `internal/ocr/ocr.go` (interface), `internal/ocr/cli.go`, `internal/ocr/gosseract.go`
- Pattern: Strategy selected at startup via `OCR_BACKEND` env var and build tags

**`ocr.Word`:**
- Purpose: Carry recognized text, confidence, and pixel bbox (`x0, y0, x1, y1`) used by erase/render
- Examples: `internal/ocr/ocr.go`
- Pattern: Plain struct; consumed by both API serialization and edit pipeline

**Document store:**
- Purpose: Track per-document page images and word lists for the duration of a session
- Examples: `internal/api/handlers.go`
- Pattern: Mutex-protected map keyed by doc id (no persistence; process-local)

## Entry Points

**Server:**
- Location: `cmd/server/main.go`
- Triggers: OS process start (manual or container)
- Responsibilities: Read env config, select OCR backend (CLI default; gosseract requires the `gosseract` build tag), start `net/http` server

## Architectural Constraints

- **Threading:** Go's `net/http` serves each request in its own goroutine; the in-memory document map must be mutex-protected (planned in `internal/api/handlers.go`). No worker pool or queue is planned.
- **Global state:** The document store is process-local singleton state in `internal/api/handlers.go`; no persistence across restarts.
- **Circular imports:** None planned; dependency direction is `cmd/server` -> `internal/api` -> `internal/{pdfio,ocr,edit}` -> external libs.
- **CGO gating:** The gosseract backend is isolated behind `//go:build gosseract` in `internal/ocr/gosseract.go` so the default build needs no CGO headers.
- **Raster flattening:** Each edited page becomes a raster image in the output PDF (inherent to scan -> image -> edit -> PDF); vector content is not preserved.
- **Input assumption:** Input PDFs are scanned (each page has an embedded raster image). Vector-page rasterization is explicitly out of scope.

## Anti-Patterns

### In-memory-only document state

**What happens:** Document records and edited images are held in process memory / local disk with no database or persistence layer.
**Why it's wrong:** State is lost on restart; horizontal scaling is impossible; cleanup depends on explicit `DELETE`.
**Do this instead:** Acceptable for the planned single-process, session-scoped tool; if persistence is later required, introduce a store interface in `internal/api` and back it with disk/DB without changing handlers.

### Optional CGO backend behind a build tag

**What happens:** `internal/ocr/gosseract.go` only compiles with `//go:build gosseract`, while the default CLI backend shells out to the Tesseract binary.
**Why it's wrong:** Two code paths can drift; gosseract variant requires a CGO toolchain and dev headers.
**Do this instead:** Keep both behind the single `ocr.Engine` interface in `internal/ocr/ocr.go` and select via `OCR_BACKEND`; document build instructions in README.md.

## Error Handling

**Strategy:** Standard Go error returns propagated up to HTTP handlers; handlers translate to HTTP status codes and JSON error bodies (exact shape not yet specified in the plan).

**Patterns:**
- Library calls (pdfcpu, gg, os/exec for Tesseract) return errors that handlers surface
- File/IO errors on missing uploads or unknown doc ids map to 4xx/5xx

## Cross-Cutting Concerns

**Logging:** Standard library `log` package expected (no structured logging dependency in `go.mod`)
**Validation:** Request body/param validation in `internal/api/handlers.go` (details TBD at implementation)
**Authentication:** None planned (no auth layer in design)

---

*Architecture analysis: 2026-10-27*
