# Integrations

**Project:** OCR PDF Text-Editing Backend (Go)
**Module:** `ocrpdf`
**Researched:** 2026-10-27
**Overall confidence:** HIGH

## Overview

This backend has no network services, databases, or cloud dependencies. All integrations are either in-process Go libraries, a local Tesseract binary, or the local filesystem. HTTP is the only external interface (REST + JSON + PNG + PDF).

## External Integrations

### 1. Tesseract OCR (local binary) - default backend

- **Type:** Subprocess (CLI)
- **Binary:** `TESSERACT_PATH` (default `C:\Program Files\Tesseract-OCR\tesseract.exe`, not on PATH)
- **Invocation:** `tesseract <image> stdout tsv` (discrete args via `os/exec.Command`; never shell)
- **Output:** 12-column TSV; level-5 rows carry `left/top/width/height/conf/text`. Confidence `-1` rows are non-text and must be filtered.
- **Trained data:** `eng.traineddata` + `osd.traineddata` under `TESSDATA_PREFIX` (set programmatically if unset).
- **Version:** 5.5.0.20241111 (Leptonica 1.85.0).
- **Used by:** `internal/ocr/cli.go` (default `OCR_BACKEND=cli`).
- **Failure modes:** missing binary (path wrong), missing tessdata (TESSDATA_PREFIX), timeout on huge images, locale/whitespace TSV variations.

### 2. gosseract/v2 (optional CGO backend)

- **Type:** In-process CGO wrapper around libtesseract
- **Module:** `github.com/otiai10/gosseract/v2` v2.4.1
- **API used:** `NewClient()`, `SetImage(path)`, `SetTessdataPrefix(prefix)`, `GetBoundingBoxesVerbose()` -> `[]BoundingBox{Box image.Rectangle, Word, Confidence, ...}`.
- **Build gate:** `//go:build gosseract` in `internal/ocr/gosseract.go`; requires CGO + Tesseract/Leptonica dev headers (absent on this machine).
- **Used by:** `internal/ocr/gosseract.go` (when `OCR_BACKEND=gosseract` and built with tag).
- **Notes:** Map `BoundingBox.Box` (Min/Max) to internal `Word{x0,y0,x1,y1}`; `Word`/`Confidence` fields. `GetBoundingBoxesVerbose` uses TSV-style output so columns align with CLI backend.

### 3. pdfcpu (in-process PDF library)

- **Type:** In-process Go library
- **Module:** `github.com/pdfcpu/pdfcpu` v0.16.3
- **API used:**
  - Extraction: `api.ExtractImagesFile(ctx, inFile, outDir, selectedPages, conf)` - writes embedded images to `outDir` with generated names; returns only error.
  - Rebuild: `api.ImportImagesFile(ctx, imgFiles, outFile, imp, conf)` - creates a PDF with one image per page.
  - Config: `pdfcpu.DefaultImportConfig()` -> `*pdfcpu.Import` (`PageDim`, `PageSize`, `Pos`, `Scale`, `InpUnit`, `BgColor`, `DPI`, ...).
  - Context: `model.Configuration` via `api.LoadConfiguration` or defaults.
- **Caveats:** pdfcpu may emit multiple images per page (soft masks, thumbnails, OCG layers); after extraction enumerate `outDir`, group by page, pick largest by pixel area. Clean `outDir` per document to avoid stale files.
- **Used by:** `internal/pdfio/extract.go`, `internal/pdfio/build.go`.

### 4. fogleman/gg + freetype + x/image (in-process rendering)

- **Type:** In-process Go libraries
- **Modules:** `github.com/fogleman/gg` v1.3.0, `github.com/golang/freetype` (indirect, truetype), `golang.org/x/image` v0.47.0.
- **API used:** `gg.NewContextForImage`, `dc.LoadFontFace(path, points)`, `dc.MeasureString`, `dc.DrawRectangle`, `dc.Fill`, `dc.DrawString`, `dc.SavePNG` / `EncodePNG`; `truetype.Parse`, `truetype.NewFace`; `image`, `image/color`, `image/draw`.
- **Used by:** `internal/edit/erase.go`, `internal/edit/render.go`.
- **Notes:** gg `font.Face` objects are not thread-safe; create per operation. `LoadFontFace` reads the TTF from `FONT_PATH`.

## Data Storage

### Databases

- **None.** Document state lives in an in-memory, mutex-protected map in `internal/api/handlers.go`. No SQL/NoSQL. Out of scope per project constraints.

### File Storage (local filesystem only)

Planned layout under a per-document working directory:

| Artifact | Description |
|----------|-------------|
| Uploaded PDF | Original input (`<docId>/input.pdf`) |
| Extracted page images | Per-page PNG/JPEG from pdfcpu (`<docId>/pages/*.png`) |
| Edited page images | After erase+render (`<docId>/edited/*.png`) |
| Result PDF | Rebuilt output (`<docId>/result.pdf`) |

- No object storage, no cloud buckets, no CDN.
- Cleanup on `DELETE /api/documents/{id}` removes the directory.
- Concurrency: `net/http` serves each request in its own goroutine; store protected by mutex. No worker pool/queue planned (sync OCR on upload).

### Caching

- None. Rebuilt PDF is regenerated on each `GET /result.pdf`. (Performance: cache + invalidate on edit is a recommended improvement.)

## HTTP API (planned)

Standard library `net/http` (enhanced `ServeMux`, Go 1.22+ routing). JSON request/response via `encoding/json`; PNG and PDF streamed with proper `Content-Type`.

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/documents` | Upload scanned PDF; extract + OCR; return `{docId, pages:[{page, words:[{text, conf, bbox}]}]}` |
| GET | `/api/documents/{id}/pages/{n}.png` | Serve page PNG for preview |
| POST | `/api/documents/{id}/edits` | Replace word(s) on a page (`oldText` or `wordIds`, `newText`, optional font/size/color) |
| GET | `/api/documents/{id}/result.pdf` | Rebuild and stream final PDF |
| DELETE | `/api/documents/{id}` | Remove document + artifacts |

- No authentication/authorization (out of scope). Use unguessable UUID doc IDs.
- No webhooks/callbacks (synchronous request/response only).
- No rate limiting or input size caps planned (security gap - see concerns).

## Authentication & Identity

- **None.** No auth provider, no sessions, no tokens. Any client can act on any doc ID. Recommended: UUID doc IDs + optional scoped tokens if exposed beyond localhost.

## Configuration (Environment Variables)

| Var | Default (Windows) | Purpose |
|-----|-------------------|---------|
| `TESSERACT_PATH` | `C:\Program Files\Tesseract-OCR\tesseract.exe` | Full path to tesseract binary |
| `TESSDATA_PREFIX` | (set programmatically if unset) | Tessdata directory |
| `OCR_BACKEND` | `cli` | `cli` or `gosseract` (latter needs `-tags gosseract` binary) |
| `FONT_PATH` | `C:\Windows\Fonts\arial.ttf` | TTF used by gg for rendering |
| `HTTP_ADDR` / port | (not specified yet) | Listen address (recommend `:8080`, configurable) |
| `DATA_DIR` / storage root | (recommended) | Root for per-document artifacts |

No secrets required; no `.env` files or secret managers detected.

## Monitoring & Observability

- **Error tracking:** None.
- **Logs:** Standard library `log` expected; recommend `log/slog` (structured) with request IDs.
- **Health endpoint:** None planned (recommend `GET /healthz`).
- **Metrics/tracing:** None.

## CI/CD & Deployment

- **Hosting:** Not specified (no deployment manifests). Single binary + Tesseract installed on host.
- **CI:** None detected. Recommended: `go build ./...`, `go vet`, `go test ./...`, plus a `-tags gosseract` build job (requires headers) and an E2E pipeline test with fixture scanned PDF.
- **Build artifacts:** `go build -o server ./cmd/server` (default); `go build -tags gosseract -o server-gosseract ./cmd/server`.

## Webhooks & Callbacks

- **Incoming:** None.
- **Outgoing:** None.

## Integration Risks (from concerns audit)

- **gosseract unbuildable on this machine** - missing `tesseract_capi.h`; keep CLI default, document header provisioning, add CI build with tag.
- **pdfcpu API drift / filename coupling** - wrap behind `internal/pdfio`; enumerate extracted files; pick largest by pixel area.
- **Tesseract subprocess safety** - use `exec.Command` with discrete args; validate resolved binary path; sanitize filenames.
- **Path traversal** - server-generated UUID doc IDs; never accept user path segments; `filepath.Clean` + root containment checks.
- **Input validation** - enforce max upload size, PDF magic bytes, page/image dimension caps, OCR timeouts.

## Sources

- `.planning/codebase/{STACK,ARCHITECTURE,INTEGRATIONS,CONCERNS}.md`
- `.planning/PROJECT.md`
- pkg.go.dev verification of pdfcpu v0.16.3, gosseract v2.4.1, gg v1.3.0, x/image v0.47.0 (2026-10-27)
- Local environment: Go 1.26.5, Tesseract 5.5.0.20241111, Arial TTF present, no Tesseract dev headers
