---
last_mapped_commit: 4de1636fce380884c3ca89be5e27619dc1d60fc8
last_mapped_at: 2026-10-27
---
# Codebase Structure

**Analysis Date:** 2026-10-27

## Current State

As of 2026-10-27 the repository has no Go source tree: only `go.mod`, `go.sum`, `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`, and `.planning/` exist. The layout below is the **planned/target structure** from that design plan; none of the listed source paths exist yet.

## Directory Layout

```
OCR_System_Go/
├── go.mod                          # module ocrpdf, Go 1.26.5
├── go.sum                          # dependency checksums
├── README.md                       # (planned) setup, endpoints, gosseract build-tag instructions
├── cmd/
│   └── server/
│       └── main.go                 # (planned) env config + HTTP server startup
└── internal/
    ├── pdfio/
    │   ├── extract.go              # (planned) pdfcpu: extract per-page images
    │   └── build.go                # (planned) pdfcpu: images -> new PDF
    ├── ocr/
    │   ├── ocr.go                  # (planned) Engine interface + Word type
    │   ├── cli.go                  # (planned) default Tesseract TSV backend
    │   └── gosseract.go            # (planned) gosseract/v2 backend (gosseract build tag)
    ├── edit/
    │   ├── erase.go                # (planned) background sampling + fill
    │   └── render.go               # (planned) gg text rendering, font fit
    └── api/
        └── handlers.go             # (planned) REST handlers + in-memory doc store
```

## Directory Purposes

**`cmd/`:**
- Purpose: Application entrypoints (Go convention: `cmd/<name>/main.go`)
- Contains: `server/main.go`
- Key files: `cmd/server/main.go`

**`cmd/server/`:**
- Purpose: Process bootstrap
- Contains: `main.go`
- Key files: `cmd/server/main.go`

**`internal/`:**
- Purpose: Private application packages not importable by external modules
- Contains: `pdfio`, `ocr`, `edit`, `api`
- Key files: package-level `.go` files above

**`internal/pdfio/`:**
- Purpose: PDF <-> raster image conversion (input extraction and output rebuild)
- Contains: `extract.go`, `build.go`
- Key files: `internal/pdfio/extract.go`, `internal/pdfio/build.go`

**`internal/ocr/`:**
- Purpose: OCR abstraction and backends
- Contains: `ocr.go`, `cli.go`, `gosseract.go`
- Key files: `internal/ocr/ocr.go`

**`internal/edit/`:**
- Purpose: In-place erase + render of replacement text on page images
- Contains: `erase.go`, `render.go`
- Key files: `internal/edit/erase.go`, `internal/edit/render.go`

**`internal/api/`:**
- Purpose: HTTP handlers and in-memory document store
- Contains: `handlers.go`
- Key files: `internal/api/handlers.go`

**`.planning/`:**
- Purpose: GSD planning artifacts (codebase maps, phases)
- Contains: `codebase/*.md`
- Key files: `.planning/codebase/*.md`
- Generated: No
- Committed: Yes

**`.zcode/`:**
- Purpose: Design plan artifacts
- Contains: `plans/*.md`
- Key files: `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`
- Generated: No
- Committed: (unknown)

## Key File Locations

**Entry Points:**
- `cmd/server/main.go`: process bootstrap and HTTP server (planned)

**Configuration:**
- `go.mod`: module declaration (`ocrpdf`, Go 1.26.5)
- `go.sum`: dependency checksums
- Env vars read in `cmd/server/main.go` (planned): `TESSERACT_PATH`, `TESSDATA_PREFIX`, `OCR_BACKEND`, `FONT_PATH`

**Core Logic:**
- `internal/api/handlers.go`: REST handlers + document store (planned)
- `internal/pdfio/extract.go`: extract page images (planned)
- `internal/pdfio/build.go`: rebuild PDF (planned)
- `internal/ocr/ocr.go`: `Engine` interface + `Word` (planned)
- `internal/ocr/cli.go`: Tesseract TSV backend (planned)
- `internal/ocr/gosseract.go`: CGO backend (planned)
- `internal/edit/erase.go`: erase (planned)
- `internal/edit/render.go`: render (planned)

**Testing:**
- Not yet present. Planned: unit tests for TSV parsing (`internal/ocr`) and font-fit (`internal/edit`), plus an end-to-end pipeline test (location TBD, likely alongside packages or under `internal/...`)

## Naming Conventions

**Files:**
- Lowercase, single-word or short snake-free names: `extract.go`, `build.go`, `handlers.go`, `erase.go`, `render.go`, `ocr.go`, `cli.go`, `gosseract.go`
- Entry point: `cmd/server/main.go`

**Directories:**
- Lowercase single words, often abbreviated domain: `pdfio`, `ocr`, `edit`, `api`
- Standard Go layout: `cmd/`, `internal/`

**Packages:**
- Match directory name: package `pdfio`, `ocr`, `edit`, `api`, `main` (for server)

**Types/Functions:**
- Not yet observed (no source). Go convention: exported identifiers in PascalCase (e.g., `Engine`, `Word`, `Recognize`), unexported in camelCase.

## Where to Add New Code

**New Feature (e.g., a new endpoint):**
- Primary code: `internal/api/handlers.go` (handler) + route registration in `cmd/server/main.go`
- Tests: co-located `*_test.go` next to the package (Go convention)

**New Component/Module:**
- Implementation: new package under `internal/<name>/` (e.g., `internal/storage/`), imported by `internal/api`

**Utilities:**
- Shared helpers: prefer adding to the relevant existing package; cross-cutting helpers could go in `internal/util/` if introduced (not currently planned)

**New OCR backend:**
- Implementation: new file in `internal/ocr/` implementing `ocr.Engine` (interface in `internal/ocr/ocr.go`)

**New edit operation:**
- Implementation: new file in `internal/edit/`

## Special Directories

**`.planning/`:**
- Purpose: GSD planning and codebase map
- Generated: Yes (by GSD tooling)
- Committed: Yes

**`.zcode/`:**
- Purpose: Design plan storage
- Generated: No
- Committed: Unknown

**`internal/`:**
- Purpose: Go-enforced private packages
- Generated: No
- Committed: Yes (once created)

---

*Structure analysis: 2026-10-27*
