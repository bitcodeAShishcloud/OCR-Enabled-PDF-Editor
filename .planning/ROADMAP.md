# Roadmap: OCR PDF Text-Editing Backend (Go)

## Overview

A Go HTTP backend that turns a scanned PDF into editable page images: extract embedded rasters, OCR words with bounding boxes, replace a detected name in place, and rebuild the final PDF. Five phases move from CGO-free bootstrap and the two high-risk pdfcpu/OCR touch points, through the in-memory document store and upload/preview/cleanup API, to the edit-and-rebuild flow that delivers the core value, verified end-to-end by a synthetic OCR round-trip.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work

- [ ] **Phase 1: Project Bootstrap & Configuration** - CGO-free server skeleton, env config, OCR engine interface, backend selection
- [ ] **Phase 2: PDF I/O Layer** - Lossless embedded-raster extraction and dimension-preserving PDF rebuild via pdfcpu
- [ ] **Phase 3: OCR Engine & TSV Parsing** - CLI (and gosseract-tagged) backend returning word text, confidence, and bboxes
- [ ] **Phase 4: In-Memory Document Store & Upload/Preview/Cleanup API** - Upload a scanned PDF, get docId + per-page OCR words, preview pages, clean up
- [ ] **Phase 5: Edit, Rebuild & End-to-End Verification** - Replace a detected word in place and download the rebuilt PDF, proven by synthetic E2E

## Phase Details

### Phase 1: Project Bootstrap & Configuration

**Goal**: The service boots as a CGO-free HTTP server, loads configuration from environment variables, validates that Tesseract and the font exist, and exposes an OCR engine interface with selectable CLI/gosseract backends.
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: CONF-01, CONF-02, OCR-01, OCR-06
**Success Criteria** (what must be TRUE):

  1. The server starts an HTTP listener from `cmd/server` and shuts down gracefully on signal.
  2. `TESSERACT_PATH`, `FONT_PATH`, and `OCR_BACKEND` are read from the environment with documented defaults, and invalid values fail fast with clear messages.
  3. The default build succeeds with `CGO_ENABLED=0`; the gosseract-tagged build succeeds separately behind `//go:build gosseract`.
  4. The `ocr.Engine` interface (`Recognize(imagePath) ([]Word, error)`) is defined and backend selection returns the configured implementation.

**Plans**: 1/1 plans executed

- [x] 01-01-PLAN.md

### Phase 2: PDF I/O Layer

**Goal**: Given a scanned PDF, extract the largest embedded raster image per page; given a set of page images, rebuild a PDF with one page per image and dimensions taken from the source images.
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: PDF-01, PDF-02, PDF-03, PDF-04
**Success Criteria** (what must be TRUE):

  1. Extraction of a scanned-PDF fixture yields one PNG per page, selecting the largest embedded image by pixel area.
  2. Rebuild from a set of page images produces a PDF with one page per image whose page dimensions match the source images (no A4 stretch).
  3. Rebuild truncates any pre-existing output file so no pages are appended.
  4. Extraction handles pages yielding multiple XObjects (masks/thumbnails) by selecting the maximum pixel area.

**Plans**: 1 plan

- [ ] 02-01-PLAN.md â€” PDF I/O layer: extract largest image per page and dimension-preserving rebuild

### Phase 3: OCR Engine & TSV Parsing

**Goal**: OCR an image to word-level text with confidence and bounding boxes through the CLI backend, with the gosseract backend available behind a build tag.
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: OCR-02, OCR-03, OCR-04, OCR-05, TEST-01
**Success Criteria** (what must be TRUE):

  1. The CLI backend runs Tesseract TSV and returns words with text, confidence, and x0/y0/x1/y1 bounding boxes.
  2. The TSV parser skips the header and blank lines, ignores rows with confidence -1, and handles variable field counts.
  3. `TESSERACT_PATH` overrides the binary location (Windows default) and `TESSDATA_PREFIX` is set when unset.
  4. Unit tests for TSV parsing pass on golden fixtures (header, empty text, -1 rows, blank lines).
  5. The gosseract backend compiles behind `//go:build gosseract` and implements the same interface.

**Plans**: 1/1 plans executed

- [x] 03-01-PLAN.md

### Phase 4: In-Memory Document Store & Upload/Preview/Cleanup API

**Goal**: A client can upload a scanned PDF and receive a document id with per-page word-level OCR results, preview any page as a PNG, and clean up the document and its artifacts.
**Mode:** mvp
**Depends on**: Phase 2, Phase 3
**Requirements**: API-01, API-02, API-05, API-06, API-07, TEST-04
**Success Criteria** (what must be TRUE):

  1. `POST /api/documents` with a scanned PDF returns a server-generated docId and per-page words with bounding boxes.
  2. `GET /api/documents/{id}/pages/{n}` serves the page PNG with the correct content type.
  3. `DELETE /api/documents/{id}` removes the document and its artifacts; later access returns 404.
  4. Upload validates the PDF magic bytes and enforces size/page caps; the in-memory store is mutex-protected.
  5. The race detector passes on the store and handlers (`go test -race`).

**Plans**: 1/1 plans executed

- [x] 04-01-PLAN.md

### Phase 5: Edit, Rebuild & End-to-End Verification

**Goal**: A client can replace a detected word on a page with new text in place and download a rebuilt PDF, with the replacement visually consistent and OCR-verifiable.
**Mode:** mvp
**Depends on**: Phase 4
**Requirements**: EDIT-01, EDIT-02, EDIT-03, EDIT-04, EDIT-05, EDIT-06, API-03, API-04, TEST-02, TEST-03
**Success Criteria** (what must be TRUE):

  1. `POST /api/documents/{id}/edits` by `oldText` or `wordIds` erases the original glyphs and renders the replacement text fitted to the bbox, returning an edited preview.
  2. Erase samples the background color from the bbox ring and fills the padded bbox; render auto-fits font size, samples text color, and baseline-aligns to the bbox bottom.
  3. The TTF font loads from `FONT_PATH` (default `C:\Windows\Fonts\arial.ttf`) and font-fit unit tests pass.
  4. `GET /api/documents/{id}/result.pdf` rebuilds one page per image (edited where edited) and streams the PDF as an attachment with correct page dimensions.
  5. The synthetic end-to-end test passes: gg draws a name, pdfcpu imports to PDF, the pipeline OCRs and replaces, and the result PDF OCRs to the new text.

**Plans**: 1/1 plans executed

- [x] 05-01-PLAN.md

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Project Bootstrap & Configuration | 1/1 | In Progress | - |
| 2. PDF I/O Layer | 0/TBD | Not started | - |
| 3. OCR Engine & TSV Parsing | 1/1 | In Progress | - |
| 4. In-Memory Document Store & Upload/Preview/Cleanup API | 1/1 | In Progress | - |
| 5. Edit, Rebuild & End-to-End Verification | 1/1 | In Progress|  |

### Phase 6: React Web UI

**Goal:** React 18 + TypeScript + Vite + Tailwind frontend consuming the Go REST API; Go serves the built frontend as static files with SPA fallback.
**Requirements**: (frontend consumes API-01..API-05, CONF-01; no new backend requirements)
**Depends on:** Phase 5
**Plans:** 1/1 plans executed

Plans:
- [x] 06-01-PLAN.md - Scaffold Vite/React/TS/Tailwind web/, implement upload/document/edit/download flows, add Go spaHandler with WEB_DIST override and SPA fallback
