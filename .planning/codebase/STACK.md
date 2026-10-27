# Technology Stack

**Analysis Date:** 2026-10-27

## Languages

**Primary:**
- Go 1.26.5 - Module `ocrpdf`; all application code (planned under `cmd/` and `internal/`)

**Secondary:**
- C/C++ (CGO) - Only required when building the `gosseract` build-tag variant of the OCR backend (`github.com/otiai10/gosseract/v2` bundles `tessbridge.cpp/.h`)

## Runtime

**Environment:**
- Go toolchain 1.26.5 (`go version go1.26.5 windows/amd64`)
- OS target observed: Windows (dev); Linux noted in plan for CGO header provisioning

**Package Manager:**
- Go modules
- Manifest: `go.mod`
- Lockfile: `go.sum` (present)

## Frameworks

**Core:**
- Go standard library (planned) - `net/http` for the REST server; no third-party web framework declared in `go.mod`

**Testing:**
- Go built-in `testing` package (planned) - Unit tests for TSV parsing and font-fit; end-to-end pipeline test

**Build/Dev:**
- Go toolchain `go build` / `go test`
- Optional build tag `gosseract` to compile the CGO-backed OCR backend (`//go:build gosseract`)

## Key Dependencies

**Critical:**
- `github.com/pdfcpu/pdfcpu` v0.16.3 - PDF text/image extraction (`api.ExtractImagesFile`) and PDF rebuild (`api.ImportImagesFile`); core to both input and output
- `github.com/otiai10/gosseract/v2` v2.4.1 - OCR backend (word-level text + bounding boxes via `GetTextBoxes`), guarded behind `gosseract` build tag
- `github.com/fogleman/gg` v1.3.0 - 2D raster rendering: erasing (fill rect) and drawing replacement text with auto-fit font size

**Infrastructure:**
- `github.com/golang/freetype` v0.0.0-20170609003504-e2365dfdc4a0 (indirect) - Font rendering support used by `gg`
- `golang.org/x/image` v0.47.0 (indirect) - Image manipulation primitives

**External binaries (not Go modules):**
- Tesseract OCR v5.5.0.20241111 - Installed at `C:\Program Files\Tesseract-OCR\tesseract.exe`; invoked as `tesseract img stdout tsv` by the default CLI OCR backend; tessdata at `C:\Program Files\Tesseract-OCR\tessdata` (eng.traineddata, osd.traineddata)
- Leptonica 1.85.0 - Bundled with the Tesseract Windows install (image I/O for Tesseract)

## Configuration

**Environment:**
- `TESSERACT_PATH` - Full path to the tesseract binary (default `C:\Program Files\Tesseract-OCR\tesseract.exe`)
- `TESSDATA_PREFIX` - Set programmatically if unset (points at the tessdata directory)
- `OCR_BACKEND` - Selects OCR implementation: `cli` (default) or `gosseract` (requires binary built with `gosseract` tag)
- `FONT_PATH` - Path to a TTF font (default `C:\Windows\Fonts\arial.ttf`)
- Additional likely: HTTP listen port (not yet specified in code; no source present)

**Build:**
- `go.mod` / `go.sum`
- Build tags: `gosseract` (enables CGO OCR backend)
- CGO requirements for gosseract variant: Linux `apt install libtesseract-dev libleptonica-dev`; Windows requires provisioning headers + CGO flags

**Files present:**
- `go.mod`, `go.sum`
- No `.env` or `.env.*` files detected
- No lint/format config detected (no `.golangci.yml`, no `gofmt` config beyond defaults)

## Platform Requirements

**Development:**
- Go 1.26.5
- Tesseract OCR 5.5.0 installed at the default Windows path (or override via `TESSERACT_PATH`)
- Arial TTF available at `C:\Windows\Fonts\arial.ttf` (or override via `FONT_PATH`)
- For gosseract builds: CGO toolchain + Tesseract/Leptonica dev headers

**Production:**
- Not yet specified (no deployment config present)
- Plan notes scanned-PDF input (embedded raster images); vector-page rasterization is out of scope

---

*Stack analysis: 2026-10-27*
