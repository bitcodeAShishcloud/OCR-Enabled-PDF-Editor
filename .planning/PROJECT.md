# OCR PDF Text-Editing Backend (Go)

## What This Is

An HTTP backend that takes a scanned PDF, extracts per-page embedded raster images with pdfcpu, OCRs them for word-level text plus bounding boxes, lets a user replace a detected name with new text drawn in place, and rebuilds the final PDF. Module `ocrpdf`. Built in Go for the Windows environment (Tesseract 5.5.0 installed, Go toolchain available).

## Core Value

Given a scanned PDF containing a name, the user can replace that name with new text in-place and get back a rebuilt PDF where the replacement is visually consistent (matching background and text color) and the new text OCRs correctly.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Upload a scanned PDF and receive a document id with per-page word-level OCR results (text, confidence, bounding boxes)
- [ ] Serve a page PNG for preview
- [ ] Replace a detected word on a page with new text (by oldText or wordIds), erasing the original and rendering the new text in place
- [ ] Rebuild and download the final PDF from the edited page images
- [ ] Clean up a document and its artifacts
- [ ] Support two OCR backends behind a common interface: CLI (Tesseract TSV, default) and gosseract/v2 (build-tagged)
- [ ] Erase original text by sampling background color from the bbox ring and filling the padded bbox
- [ ] Render replacement text with auto-fitted font size, sampled text color, baseline-aligned to the bbox
- [ ] Configurable Tesseract path, font path, OCR backend via environment variables

### Out of Scope

- Vector-page rasterization — only scanned PDFs with embedded raster images are supported; vector pages are documented as out of scope
- Non-Windows primary target — Windows is the primary dev/runtime environment; Linux/macOS supported via env vars and documented build steps
- Authentication/authorization — no user accounts; documents held in an in-memory store with mutex protection
- Persistent storage — documents are in-memory only; no database
- Multi-tenant isolation — single-process, in-memory document store

## Context

- Existing Go module `ocrpdf` with `go.mod`/`go.sum` declaring dependencies: pdfcpu v0.16.3, gosseract/v2 v2.4.1 (indirect), gg v1.3.0 (indirect), golang.org/x/image v0.47.0 (indirect), freetype (indirect)
- Codebase map produced in `.planning/codebase/` (architecture is planned; no Go source yet)
- Detailed implementation plan captured in `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`
- Tesseract 5.5.0 installed at `C:\Program Files\Tesseract-OCR\tesseract.exe` (not on PATH)
- Default font `C:\Windows\Fonts\arial.ttf`
- Synthetic end-to-end test: gg draws a name onto an image, pdfcpu imports it to a PDF, full pipeline OCRs and replaces, resulting PDF OCRs to the new text
- Unit tests for TSV parsing and the font-fit function

## Constraints

- **Tech stack**: Go 1.26.5, pdfcpu v0.16.3 for extraction and rebuild, fogleman/gg for rendering, otiai10/gosseract/v2 behind build tag, golang.org/x/image
- **Compatibility**: default build must not require CGO headers; gosseract backend guarded by `//go:build gosseract`
- **Environment**: Windows primary; TESSERACT_PATH, FONT_PATH, OCR_BACKEND configurable via env vars
- **Architecture**: flattened raster pipeline — each edited page becomes a raster image in the output PDF (inherent to scan→image→edit→PDF)
- **Concurrency**: in-memory document store protected by mutex

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| OCR backend: both CLI (default) and gosseract (build-tagged) behind `ocr.Engine` interface | CLI works with installed Tesseract immediately; gosseract needs CGO headers | — Pending |
| pdfcpu for both extraction and rebuild | Lossless extraction of embedded raster images; simple image→page rebuild | — Pending |
| Erase by sampling background color from bbox ring | Robust on uniform scanned backgrounds | — Pending |
| Render by binary-searching font size to fit bbox width, baseline-aligned | Keeps replacement visually consistent with original | — Pending |
| Coarse granularity, YOLO mode, adaptive model profile | Lean phase slicing, auto-approval, cost-optimized models | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-10-27 after initialization*