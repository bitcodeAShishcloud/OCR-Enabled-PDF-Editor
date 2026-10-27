---
last_mapped_commit: 4de1636fce380884c3ca89be5e27619dc1d60fc8
last_mapped_at: 2026-10-27
---
# External Integrations

**Analysis Date:** 2026-10-27

## APIs & External Services

**OCR (local binary):**
- Tesseract OCR v5.5.0 - Word-level text recognition + bounding boxes
  - Invocation: `tesseract <image> stdout tsv` (CLI backend, default); 12-column TSV including `left/top/width/height/conf`
  - Binary path: `TESSERACT_PATH` (default `C:\Program Files\Tesseract-OCR\tesseract.exe`)
  - Trained data: `eng.traineddata` (and `osd.traineddata`) under `TESSDATA_PREFIX`
  - SDK/Client: `github.com/otiai10/gosseract/v2` (optional CGO backend behind `gosseract` build tag, `GetTextBoxes`)

**PDF processing (in-process library):**
- pdfcpu v0.16.3 - Extraction of embedded per-page raster images (`api.ExtractImagesFile`) and rebuild of a PDF from images (`api.ImportImagesFile`)
  - Client: `github.com/pdfcpu/pdfcpu`
  - No network; operates on local files

**Raster graphics (in-process library):**
- fogleman/gg v1.3.0 - Erase + render pipeline (sample background color, fill padded bbox, auto-fit font, draw replacement text)
  - Client: `github.com/fogleman/gg`
  - Font: FreeType via `github.com/golang/freetype` (indirect)

## Data Storage

**Databases:**
- None - Document state held in an in-memory, mutex-protected map (planned in `internal/api`)

**File Storage:**
- Local filesystem only (planned) - Uploaded PDFs, extracted page PNGs, edited page images, rebuilt result PDF
- No object storage or cloud bucket integration detected

**Caching:**
- None

## Authentication & Identity

**Auth Provider:**
- None - No authentication/authorization layer present or planned in the design doc

## Monitoring & Observability

**Error Tracking:**
- None

**Logs:**
- Standard Go `log` package expected (no structured logging library in dependencies)

## CI/CD & Deployment

**Hosting:**
- Not specified (no deployment manifests present)

**CI Pipeline:**
- None detected (no `.github/workflows`, no CI config files)

## Environment Configuration

**Required env vars:**
- `TESSERACT_PATH` - Path to tesseract binary (has default)
- `TESSDATA_PREFIX` - Tessdata directory (set programmatically if unset)
- `OCR_BACKEND` - `cli` (default) or `gosseract`
- `FONT_PATH` - TTF font path (default `C:\Windows\Fonts\arial.ttf`)

**Secrets location:**
- No secrets management detected; no `.env` files present
- No credentials or API keys required by the current dependency set

## Webhooks & Callbacks

**Incoming:**
- None (planned REST endpoints are synchronous request/response, no webhook receivers)

**Outgoing:**
- None

---

*Integration audit: 2026-10-27*
