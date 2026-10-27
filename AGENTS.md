<!-- GSD:project-start source:PROJECT.md -->

## Project

**OCR PDF Text-Editing Backend (Go)**

An HTTP backend that takes a scanned PDF, extracts per-page embedded raster images with pdfcpu, OCRs them for word-level text plus bounding boxes, lets a user replace a detected name with new text drawn in place, and rebuilds the final PDF. Module `ocrpdf`. Built in Go for the Windows environment (Tesseract 5.5.0 installed, Go toolchain available).

**Core Value:** Given a scanned PDF containing a name, the user can replace that name with new text in-place and get back a rebuilt PDF where the replacement is visually consistent (matching background and text color) and the new text OCRs correctly.

### Constraints

- **Tech stack**: Go 1.26.5, pdfcpu v0.16.3 for extraction and rebuild, fogleman/gg for rendering, otiai10/gosseract/v2 behind build tag, golang.org/x/image
- **Compatibility**: default build must not require CGO headers; gosseract backend guarded by `//go:build gosseract`
- **Environment**: Windows primary; TESSERACT_PATH, FONT_PATH, OCR_BACKEND configurable via env vars
- **Architecture**: flattened raster pipeline — each edited page becomes a raster image in the output PDF (inherent to scan→image→edit→PDF)
- **Concurrency**: in-memory document store protected by mutex

<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->

## Technology Stack

## Languages

- Go 1.26.5 - Module `ocrpdf`; all application code (planned under `cmd/` and `internal/`)
- C/C++ (CGO) - Only required when building the `gosseract` build-tag variant of the OCR backend (`github.com/otiai10/gosseract/v2` bundles `tessbridge.cpp/.h`)

## Runtime

- Go toolchain 1.26.5 (`go version go1.26.5 windows/amd64`)
- OS target observed: Windows (dev); Linux noted in plan for CGO header provisioning
- Go modules
- Manifest: `go.mod`
- Lockfile: `go.sum` (present)

## Frameworks

- Go standard library (planned) - `net/http` for the REST server; no third-party web framework declared in `go.mod`
- Go built-in `testing` package (planned) - Unit tests for TSV parsing and font-fit; end-to-end pipeline test
- Go toolchain `go build` / `go test`
- Optional build tag `gosseract` to compile the CGO-backed OCR backend (`//go:build gosseract`)

## Key Dependencies

- `github.com/pdfcpu/pdfcpu` v0.16.3 - PDF text/image extraction (`api.ExtractImagesFile`) and PDF rebuild (`api.ImportImagesFile`); core to both input and output
- `github.com/otiai10/gosseract/v2` v2.4.1 - OCR backend (word-level text + bounding boxes via `GetTextBoxes`), guarded behind `gosseract` build tag
- `github.com/fogleman/gg` v1.3.0 - 2D raster rendering: erasing (fill rect) and drawing replacement text with auto-fit font size
- `github.com/golang/freetype` v0.0.0-20170609003504-e2365dfdc4a0 (indirect) - Font rendering support used by `gg`
- `golang.org/x/image` v0.47.0 (indirect) - Image manipulation primitives
- Tesseract OCR v5.5.0.20241111 - Installed at `C:\Program Files\Tesseract-OCR\tesseract.exe`; invoked as `tesseract img stdout tsv` by the default CLI OCR backend; tessdata at `C:\Program Files\Tesseract-OCR\tessdata` (eng.traineddata, osd.traineddata)
- Leptonica 1.85.0 - Bundled with the Tesseract Windows install (image I/O for Tesseract)

## Configuration

- `TESSERACT_PATH` - Full path to the tesseract binary (default `C:\Program Files\Tesseract-OCR\tesseract.exe`)
- `TESSDATA_PREFIX` - Set programmatically if unset (points at the tessdata directory)
- `OCR_BACKEND` - Selects OCR implementation: `cli` (default) or `gosseract` (requires binary built with `gosseract` tag)
- `FONT_PATH` - Path to a TTF font (default `C:\Windows\Fonts\arial.ttf`)
- Additional likely: HTTP listen port (not yet specified in code; no source present)
- `go.mod` / `go.sum`
- Build tags: `gosseract` (enables CGO OCR backend)
- CGO requirements for gosseract variant: Linux `apt install libtesseract-dev libleptonica-dev`; Windows requires provisioning headers + CGO flags
- `go.mod`, `go.sum`
- No `.env` or `.env.*` files detected
- No lint/format config detected (no `.golangci.yml`, no `gofmt` config beyond defaults)

## Platform Requirements

- Go 1.26.5
- Tesseract OCR 5.5.0 installed at the default Windows path (or override via `TESSERACT_PATH`)
- Arial TTF available at `C:\Windows\Fonts\arial.ttf` (or override via `FONT_PATH`)
- For gosseract builds: CGO toolchain + Tesseract/Leptonica dev headers
- Not yet specified (no deployment config present)
- Plan notes scanned-PDF input (embedded raster images); vector-page rasterization is out of scope

<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

## Naming Patterns

- Not detected - repository contains no Go source files yet (only `go.mod`, `go.sum`).
- Not detected.
- Not detected.
- Not detected.

## Code Style

- Not detected - no source files or formatter config present. Go projects conventionally use `gofmt`/`goimports`, but no configuration is committed.
- Not detected - no `.golangci.yml`, `Makefile`, or lint config present.

## Import Organization

- Not detected.
- Module path declared in `go.mod`: `ocrpdf` (Go 1.26.5). No internal packages or path aliases exist yet.

## Error Handling

- Not detected.

## Logging

- Not detected.

## Comments

- Not detected.
- Not applicable (Go codebase). Go doc comments (`// Package ...`) not yet present.

## Function Design

## Module Design

<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

## Current State

## System Overview

```text

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

- Standard library only for HTTP (`net/http`); no third-party web framework declared in `go.mod`
- OCR behind a single `ocr.Engine` interface with two implementations selected at startup via `OCR_BACKEND` and a build tag
- PDF processing is lossless extraction of embedded scanned images (vector page rasterization is out of scope)
- Stateless-per-request handlers backed by a mutex-protected in-memory document map (no database)
- Heavy use of external binaries/libraries: Tesseract (CLI or CGO), pdfcpu, fogleman/gg

## Layers

- Purpose: Configuration and process startup
- Location: `cmd/server/`
- Contains: `main.go`
- Depends on: `internal/api`, `internal/ocr`, env vars
- Used by: OS process / container entrypoint
- Purpose: HTTP routing, request validation, orchestration of pdfio/ocr/edit, document lifecycle
- Location: `internal/api/`
- Contains: `handlers.go`
- Depends on: `internal/pdfio`, `internal/ocr`, `internal/edit`
- Used by: `cmd/server/main.go`
- Purpose: Bridge between PDF files and per-page raster images (input extraction and output rebuild)
- Location: `internal/pdfio/`
- Contains: `extract.go`, `build.go`
- Depends on: `github.com/pdfcpu/pdfcpu`
- Used by: `internal/api`
- Purpose: Recognize words + bounding boxes from a page image
- Location: `internal/ocr/`
- Contains: `ocr.go` (interface + `Word`), `cli.go` (default), `gosseract.go` (optional CGO)
- Depends on: Tesseract binary or `gosseract/v2`
- Used by: `internal/api`
- Purpose: In-place text replacement on a raster page (erase old glyphs, render new text)
- Location: `internal/edit/`
- Contains: `erase.go`, `render.go`
- Depends on: `github.com/fogleman/gg`, `golang.org/x/image`, font files
- Used by: `internal/api`

## Data Flow

### Primary Request Path (upload + OCR)

### Edit Path

### Result Build Path

### Cleanup Path

### State Management

- No database; document state lives in an in-memory, mutex-protected map inside `internal/api/handlers.go`
- Page images and edited images are materialized as files on the local filesystem; the store holds references/paths

## Key Abstractions

- Purpose: Decouple OCR implementation from the API and edit layers
- Examples: `internal/ocr/ocr.go` (interface), `internal/ocr/cli.go`, `internal/ocr/gosseract.go`
- Pattern: Strategy selected at startup via `OCR_BACKEND` env var and build tags
- Purpose: Carry recognized text, confidence, and pixel bbox (`x0, y0, x1, y1`) used by erase/render
- Examples: `internal/ocr/ocr.go`
- Pattern: Plain struct; consumed by both API serialization and edit pipeline
- Purpose: Track per-document page images and word lists for the duration of a session
- Examples: `internal/api/handlers.go`
- Pattern: Mutex-protected map keyed by doc id (no persistence; process-local)

## Entry Points

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

### Optional CGO backend behind a build tag

## Error Handling

- Library calls (pdfcpu, gg, os/exec for Tesseract) return errors that handlers surface
- File/IO errors on missing uploads or unknown doc ids map to 4xx/5xx

## Cross-Cutting Concerns

<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-fast` for a trivial task inline, with no subagents and no PLAN.md
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
