# Requirements: OCR PDF Text-Editing Backend (Go)

**Defined:** 2026-10-27
**Core Value:** Given a scanned PDF containing a name, the user can replace that name with new text in-place and get back a rebuilt PDF where the replacement is visually consistent and the new text OCRs correctly.

## v1 Requirements

### PDF I/O

- [ ] **PDF-01**: Service extracts the largest embedded raster image per page from an uploaded scanned PDF using pdfcpu
- [ ] **PDF-02**: Service rebuilds a PDF from a set of page images using pdfcpu (one image per page)
- [ ] **PDF-03**: Rebuild truncates any existing output file before writing (no append)
- [ ] **PDF-04**: Rebuild sets page dimensions from the source image (no A4 stretch)

### OCR

- [x] **OCR-01**: OCR engine interface exposes `Recognize(imagePath) ([]Word, error)` with word text, confidence, and bounding box
- [ ] **OCR-02**: CLI backend parses Tesseract TSV (level-5 rows) into words with text/conf/bbox
- [ ] **OCR-03**: CLI backend locates tesseract via `TESSERACT_PATH` with Windows default
- [ ] **OCR-04**: CLI backend sets `TESSDATA_PREFIX` if unset
- [ ] **OCR-05**: gosseract backend implements the same interface behind `//go:build gosseract`
- [x] **OCR-06**: OCR backend selectable at startup via `OCR_BACKEND=cli|gosseract`

### Editing

- [ ] **EDIT-01**: Erase original text by sampling background color from the bbox ring and filling the padded bbox
- [ ] **EDIT-02**: Render replacement text with font size auto-fitted to bbox width
- [ ] **EDIT-03**: Render uses text color sampled from the original glyphs' darkest pixels
- [ ] **EDIT-04**: New text is baseline-aligned to the bbox bottom
- [ ] **EDIT-05**: Load TTF font from `FONT_PATH` (default `C:\Windows\Fonts\arial.ttf`)
- [ ] **EDIT-06**: Edit endpoint accepts page, oldText or wordIds, newText, and optional font/size/color overrides

### API

- [ ] **API-01**: `POST /api/documents` uploads PDF, extracts images, OCRs all pages, returns docId and per-page words with bboxes
- [ ] **API-02**: `GET /api/documents/{id}/pages/{n}` serves the page PNG
- [ ] **API-03**: `POST /api/documents/{id}/edits` applies an edit and returns the edited preview
- [ ] **API-04**: `GET /api/documents/{id}/result.pdf` rebuilds and streams the final PDF
- [ ] **API-05**: `DELETE /api/documents/{id}` cleans up document and artifacts
- [ ] **API-06**: In-memory document store is mutex-protected
- [ ] **API-07**: Document IDs are server-generated

### Configuration

- [x] **CONF-01**: Configuration via environment variables (`TESSERACT_PATH`, `FONT_PATH`, `OCR_BACKEND`)
- [x] **CONF-02**: Default build is CGO-free

### Testing

- [ ] **TEST-01**: Unit tests for TSV parsing
- [ ] **TEST-02**: Unit tests for font-fit function
- [ ] **TEST-03**: Synthetic end-to-end test: gg draws a name onto an image, pdfcpu imports to PDF, full pipeline OCRs and replaces, result PDF OCRs to new text
- [ ] **TEST-04**: Race detector passes on store and handlers

## v2 Requirements

### Future Enhancements

- **V2-01**: gosseract backend exercised in CI on a machine with MinGW headers
- **V2-02**: Bounded worker pool for Tesseract concurrency
- **V2-03**: Color-sampling polish and additional edit overrides (font family, alignment)
- **V2-04**: Edited-preview response shape refinement (image bytes vs URL)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Authentication/authorization | No user accounts; single-process in-memory store |
| Persistent storage / database | Documents are in-memory only |
| Vector-page rasterization | Only scanned PDFs with embedded raster images |
| Multi-tenant isolation | Single-process, in-memory document store |
| Batch edits | One edit per request in v1 |
| Cloud storage | Local filesystem only |
| Horizontal scaling | Single-process design |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| API-01 | Phase 4 | Pending |
| API-02 | Phase 4 | Pending |
| API-03 | Phase 5 | Pending |
| API-04 | Phase 5 | Pending |
| API-05 | Phase 4 | Pending |
| API-06 | Phase 4 | Pending |
| API-07 | Phase 4 | Pending |
| CONF-01 | Phase 1 | Complete |
| CONF-02 | Phase 1 | Complete |
| EDIT-01 | Phase 5 | Pending |
| EDIT-02 | Phase 5 | Pending |
| EDIT-03 | Phase 5 | Pending |
| EDIT-04 | Phase 5 | Pending |
| EDIT-05 | Phase 5 | Pending |
| EDIT-06 | Phase 5 | Pending |
| OCR-01 | Phase 1 | Complete |
| OCR-02 | Phase 3 | Pending |
| OCR-03 | Phase 3 | Pending |
| OCR-04 | Phase 3 | Pending |
| OCR-05 | Phase 3 | Pending |
| OCR-06 | Phase 1 | Complete |
| PDF-01 | Phase 2 | Pending |
| PDF-02 | Phase 2 | Pending |
| PDF-03 | Phase 2 | Pending |
| PDF-04 | Phase 2 | Pending |
| TEST-01 | Phase 3 | Pending |
| TEST-02 | Phase 5 | Pending |
| TEST-03 | Phase 5 | Pending |
| TEST-04 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 29 total
- Mapped to phases: 29
- Unmapped: 0

---
*Requirements defined: 2026-10-27*
*Last updated: 2026-10-27 after initial definition*
