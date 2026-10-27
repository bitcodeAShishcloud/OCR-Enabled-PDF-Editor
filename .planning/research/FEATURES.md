# Feature Landscape

**Domain:** Go HTTP backend for scanned-PDF OCR text replacement
**Researched:** 2026-10-27
**Overall confidence:** HIGH (derived from project plan, codebase map, and requirements; no external sources reachable)

## Table Stakes

Features users expect. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Upload scanned PDF and receive doc id + per-page word-level OCR | Core entry point; every downstream edit depends on recognized words + bboxes | High | `POST /api/documents`. Runs pdfcpu extraction + OCR over all pages synchronously; returns `{docId, pages:[{page, words:[{text, conf, bbox}]}]}` |
| Word-level OCR with bounding boxes | Replacement needs precise pixel bbox (x0,y0,x1,y1) to erase/render in place | High | `ocr.Word{text, confidence, x0, y0, x1, y1}`; Tesseract TSV level-5 rows (12 cols incl. left/top/width/height/conf) |
| Per-page preview PNG | Client must visually confirm which word to replace and inspect result | Medium | `GET /api/documents/{id}/pages/{n}` serves the extracted/edited page image |
| Replace detected word by oldText or wordIds | The primary edit operation; matching by text or explicit ids covers both UX flows | High | `POST /api/documents/{id}/edits` with `{page, oldText or wordIds, newText, font?, size?, color?}`; erases then renders |
| Rebuild and download final PDF | Deliverable artifact; flattens edited page rasters back into a PDF | Medium | `GET /api/documents/{id}/result.pdf` via pdfcpu `ImportImagesFile` (one image per page) |
| Clean up document and artifacts | Avoids unbounded disk/memory growth in a session-scoped tool | Low | `DELETE /api/documents/{id}` removes in-memory record + uploaded PDF, page images, edited images, result PDF |
| In-memory document store with mutex protection | Thread-safe state for concurrent `net/http` goroutines; no DB dependency | Medium | Process-local map keyed by doc id in `internal/api/handlers.go`; lost on restart (accepted) |
| Configurable env vars (TESSERACT_PATH, TESSDATA_PREFIX, OCR_BACKEND, FONT_PATH) | Windows Tesseract not on PATH; font path varies by OS | Low | Read in `cmd/server/main.go`; defaults to `C:\Program Files\Tesseract-OCR\tesseract.exe` and `C:\Windows\Fonts\arial.ttf` |

### REST Endpoints as Feature Set

| Endpoint | Method | Feature | Complexity |
|----------|--------|---------|------------|
| `/api/documents` | POST | Upload + extract + OCR all pages | High |
| `/api/documents/{id}/pages/{n}` | GET | Serve page PNG preview | Low |
| `/api/documents/{id}/edits` | POST | Erase + render replacement text | High |
| `/api/documents/{id}/result.pdf` | GET | Rebuild PDF from edited images | Medium |
| `/api/documents/{id}` | DELETE | Cleanup document + artifacts | Low |

## Differentiators

Features that set product apart. Not expected, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Dual OCR backend behind common interface | CLI works immediately with installed Tesseract 5.5.0; gosseract/v2 available when CGO headers present | Medium | `ocr.Engine` interface in `internal/ocr/ocr.go`; CLI default (`internal/ocr/cli.go`), gosseract behind `//go:build gosseract` (`internal/ocr/gosseract.go`); selected via `OCR_BACKEND` |
| Background sampling erase | Robust erasure on uniform scanned backgrounds without manual color picking | Medium | Sample bg color from pixel ring around bbox, fill padded rect (`internal/edit/erase.go`) |
| Font-fit render | Replacement text visually consistent with original bbox; no overflow | Medium | Binary-search font size to fit bbox width, baseline-aligned to bbox bottom (`internal/edit/render.go`) |
| Text color sampling | Matching replacement color to original glyphs | Medium | Sample darkest pixels of old glyphs for text color |
| Auto-fit with optional overrides | Users can override font/size/color when auto-fit is wrong | Low | Edit request accepts `font?, size?, color?` overrides |
| Lossless embedded-image extraction | Avoids re-encoding quality loss on input side | Medium | pdfcpu `ExtractImagesFile` pulls largest embedded raster per page |
| Synthetic end-to-end verification | Confidence that OCR detects name, edit replaces, resulting PDF OCRs to new text | Medium | gg draws name onto image, pdfcpu imports to PDF, full pipeline, then OCR verifies |
| Unit tests for TSV parsing and font-fit | Deterministic regression coverage for the two most fragile parsers | Low | Co-located `*_test.go` |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Authentication / authorization | No user accounts; adds complexity and deployment friction for a session-scoped tool | None; single-process anonymous access |
| Persistent storage / database | In-memory only keeps the tool stateless-deployable and simple | In-memory mutex-protected map + local filesystem artifacts; explicit `DELETE` cleanup |
| Vector-page rasterization | Out of scope; only scanned PDFs with embedded raster images are supported | Document as limitation; reject/ignore vector pages rather than rasterize |
| Multi-tenant isolation | Single-process tool; isolation adds overhead without value at this scope | Single global document store |
| Real-time progress / streaming edits | Synchronous request/response is simpler and sufficient for per-page edits | Return edited preview PNG in the edit response |
| Multi-page batch edit in one call | Complicates matching and partial failure handling | One page per edit request; client loops |
| Cloud object storage / CDN | Local filesystem is sufficient for session artifacts | Keep artifacts on local disk |
| Structured logging / observability stack | No dependency in `go.mod`; stdlib `log` is enough | Use standard library `log` |
| OCR language selection UI | Default `eng.traineddata`; not needed for name replacement | Fixed `eng` (configurable only via TESSDATA_PREFIX) |
| Horizontal scaling | In-memory state prevents it; accepted tradeoff | Keep single-process; if needed later, introduce store interface in `internal/api` |

## Feature Dependencies

```
Upload PDF (POST /api/documents)
  -> Extract page images (pdfcpu) -> OCR all pages (ocr.Engine) -> store doc record (id, images, word lists)
  -> Preview PNG available (GET .../pages/{n})

Preview PNG (GET .../pages/{n}) -> requires Upload

Edit (POST .../edits) -> requires Upload + OCR word list for target page
  -> Erase (background sampling) -> Render (font-fit, color sampling) -> updated preview image
  -> optional font/size/color overrides

Result PDF (GET .../result.pdf) -> requires Upload; reflects latest edited images (unedited pages pass through)

Cleanup (DELETE ...) -> requires Upload; removes record + all artifacts

Dual OCR backend -> requires ocr.Engine interface + both implementations
Font-fit render -> requires erase completed (bbox known) + TTF font loaded
Color sampling (text) -> requires old glyphs present before erase (sample before overwriting)
```

## MVP Recommendation

Prioritize:
1. Upload + OCR all pages (`POST /api/documents`) — table stakes, unblocks everything
2. Page PNG preview (`GET .../pages/{n}`) — enables user selection
3. Edit by oldText/wordIds (`POST .../edits`) with erase + font-fit render — core value
4. Rebuild/download PDF (`GET .../result.pdf`) — deliverable
5. Cleanup (`DELETE ...`) — hygiene

Defer:
- gosseract backend (behind build tag): ship CLI default first; add CGO variant once headers available
- Text color sampling polish: start with black fallback, then sample darkest glyphs
- Optional font/size/color overrides: add after auto-fit baseline works
- Synthetic end-to-end test harness: build alongside MVP to validate core value

## Sources

- `.planning/PROJECT.md` — requirements, out of scope, constraints
- `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md` — REST API table, technical decisions
- `.planning/codebase/ARCHITECTURE.md` and `STRUCTURE.md` — component responsibilities, data flow
- `.planning/codebase/INTEGRATIONS.md` — external services and env vars
- No external web sources (search unavailable); domain knowledge from project context only
