# Technology Stack

**Project:** OCR PDF Text-Editing Backend (Go)
**Module:** `ocrpdf`
**Researched:** 2026-10-27
**Overall confidence:** HIGH (versions verified against pkg.go.dev / installed toolchain)

## Recommended Stack

### Core Runtime

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go toolchain | 1.26.5 (`go1.26.5 windows/amd64`) | Compile/run; module `ocrpdf` | Declared in `go.mod`; `go` directive 1.26.5. Recent toolchain; stdlib `net/http` enhanced routing (1.22+) and experimental `encoding/json/v2` (1.25+) available. |
| OS target | Windows (primary), Linux/macOS via env overrides | Dev/runtime | Tesseract 5.5.0 installed at `C:\Program Files\Tesseract-OCR`; default font `C:\Windows\Fonts\arial.ttf`. |

### PDF Processing (in-process)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/pdfcpu/pdfcpu` | v0.16.3 (published 2026-10-19) | Extract embedded raster images (`api.ExtractImagesFile`) and rebuild PDF from images (`api.ImportImagesFile`) | Pure Go, no CGO, Apache-2.0. Supports PDF through 2.0. Lossless extraction of embedded XObjects - exactly what scanned-PDF pipeline needs. |
| `github.com/pdfcpu/pdfcpu/pkg/api` | (same) | File-based entry points used by `internal/pdfio` | `ExtractImagesFile(ctx, inFile, outDir, selectedPages, conf)` and `ImportImagesFile(ctx, imgFiles, outFile, imp, conf)`. |
| `github.com/pdfcpu/pdfcpu/pkg/pdfcpu` | (same) | `*pdfcpu.Import` config (`DefaultImportConfig`) | Controls page dimensions, DPI, position, scale, bg color for rebuilt pages. |

### OCR

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Tesseract OCR (external binary) | 5.5.0.20241111 (Leptonica 1.85.0) | Default CLI OCR backend: `tesseract <img> stdout tsv` | Installed locally; CLI backend needs no CGO headers. TSV output gives 12 columns incl. `left/top/width/height/conf` at level 5. |
| `github.com/otiai10/gosseract/v2` | v2.4.1 (published 2023-09-06) | Optional CGO OCR backend behind `//go:build gosseract` | Wraps libtesseract C++ API; `GetBoundingBoxesVerbose()` returns word-level `BoundingBox{Box image.Rectangle, Word, Confidence, BlockNum, ParNum, LineNum, WordNum}`. MIT. |
| `os/exec` (stdlib) | Go 1.26.5 | Invoke tesseract binary with discrete args | Avoids shell injection; `exec.Command(tesseractPath, img, "stdout", "tsv")`. |

### Raster Rendering (in-process)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/fogleman/gg` | v1.3.0 (published 2019-02-20) | Erase (fill padded bbox) + render replacement text; auto-fit font size; `MeasureString`, `LoadFontFace`, `SavePNG` | Pure Go, MIT, 1942 importers. Rich 2D API. Unmaintained since 2019 - acceptable behind `internal/edit` seam so migration is localized. |
| `github.com/golang/freetype` | v0.0.0-20170609003504 (truetype subpackage) | TTF parsing used by `gg.LoadFontFace` (`truetype.Parse`, `truetype.NewFace`) | Indirect via gg. Pure Go. BSD-style. |
| `golang.org/x/image` | v0.47.0 (published 2026-10-09) | `image`, `image/draw`, `font`, `math/fixed`, encoding helpers; `image/color` sampling | Official Go subrepo; BSD-3-Clause. Used directly for pixel sampling in erase/render. |
| `golang.org/x/image/font` | (same module) | `font.Face` interface consumed by gg | Standard font abstraction; keeps gg decoupled from freetype internals. |

### HTTP / JSON (stdlib only)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `net/http` (stdlib) | Go 1.26.5 | REST server, routing, middleware, file streaming | No third-party web framework needed. Enhanced `ServeMux` patterns (method + path, wildcards) since Go 1.22. |
| `encoding/json` (stdlib) | Go 1.26.5 | Request/response JSON (doc records, word lists, edit payloads) | Native, sufficient for this API. Experimental `json/v2` (1.25+) available if stricter decoding desired. |
| `encoding/csv` (stdlib) | Go 1.26.5 | Parse Tesseract TSV robustly (tab delimiter, trailing empty fields) | Handles quoted/empty fields better than `strings.Split`. |
| `log/slog` (stdlib) | Go 1.26.5 | Structured logging (recommended over `log`) | Available since 1.21; no external dep. |
| `mime/multipart` (stdlib) | Go 1.26.5 | PDF upload parsing (`multipart/form-data`) | Native; enforce `MaxBytesReader`. |

### Concurrency / State (stdlib)

| Technology | Purpose | Why |
|------------|---------|-----|
| `sync.Mutex` / `sync.RWMutex` | Protect in-memory document map | Planned single-process store; no DB. |
| `context.Context` | Request cancellation, pdfcpu API calls | pdfcpu file/io APIs accept `context.Context`. |

## External Binaries / System Dependencies

| Dependency | Version | Location / Notes |
|------------|---------|------------------|
| Tesseract OCR | 5.5.0.20241111 | `C:\Program Files\Tesseract-OCR\tesseract.exe` (not on PATH). Override via `TESSERACT_PATH`. tessdata: `eng.traineddata`, `osd.traineddata`. |
| Leptonica | 1.85.0 | Bundled with Tesseract Windows install (image I/O). |
| Arial TTF | Windows system font | `C:\Windows\Fonts\arial.ttf`. Override via `FONT_PATH`. |
| Tesseract dev headers (`tesseract_capi.h`, libleptonica) | Not installed | Required only for `gosseract` build tag. Absent on this machine => gosseract backend cannot be built/tested locally. |

## CGO Requirements

| Build variant | CGO | Headers needed | How to build |
|---------------|-----|----------------|--------------|
| Default (CLI OCR) | `CGO_ENABLED=0` (or default with no CGO imports) | None | `go build ./...` / `go test ./...` - pure Go, works on Windows with stock Go. |
| `gosseract` tag | `CGO_ENABLED=1` | Tesseract + Leptonica dev headers (`tesseract_capi.h`, `leptonica/*.h`) and a C++ toolchain | `go build -tags gosseract ./...`. On Linux: `apt install libtesseract-dev libleptonica-dev`. On Windows: provision headers + set `CGO_CFLAGS`/`CGO_LDFLAGS` (e.g. via MSYS2/MinGW) - not currently installed here. |

**Gating rule:** `internal/ocr/gosseract.go` must carry `//go:build gosseract` so the default build never pulls CGO. The `ocr.Engine` interface in `internal/ocr/ocr.go` keeps both backends interchangeable; select via `OCR_BACKEND=cli|gosseract`.

## Windows-Specific Notes

- Default paths are Windows-only (`C:\Program Files\Tesseract-OCR\tesseract.exe`, `C:\Windows\Fonts\arial.ttf`). Resolve via env vars and fall back to PATH lookup before the hardcoded default for cross-platform safety.
- `TESSDATA_PREFIX` is set programmatically if unset (point at the tessdata directory). Tesseract 5.5 reads it.
- File path handling: use `filepath` (not `path`) and `os.PathSeparator`; sanitize doc IDs to prevent traversal.
- Process spawning: `exec.Command` with discrete args; quote/validate image paths; avoid `sh -c`.
- PNG/image I/O: pdfcpu writes extracted images to `outDir` with generated names; enumerate + pick largest by pixel area per page (pdfcpu may emit masks/thumbnails).

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| PDF library | pdfcpu v0.16.3 | unidoc/unipdf | Commercial license; heavier; not needed for raster extraction/rebuild. |
| PDF library | pdfcpu v0.16.3 | go-fitz (MuPDF bindings) | CGO + MuPDF licensing; overkill; pdfcpu pure Go. |
| OCR | Tesseract CLI + gosseract (build-tagged) | Pure-Go OCR (gosseract only / tesseract.js) | No mature pure-Go Tesseract; CLI reuses installed binary. |
| OCR backend | CLI default | gosseract default | gosseract needs CGO headers absent on dev machine; CLI works immediately. |
| Rendering | fogleman/gg v1.3.0 | `golang.org/x/image/draw` + freetype direct | More boilerplate; gg provides MeasureString/LoadFontFace/Fill; keep behind `internal/edit` for future swap. |
| Rendering | fogleman/gg v1.3.0 | Ebiten / imaging libs | Heavier; not needed for 2D raster text. |
| HTTP framework | `net/http` stdlib | Gin / Chi / Fiber | No need; stdlib enhanced ServeMux (1.22+) covers routing; fewer deps. |
| JSON | `encoding/json` | json-iterator / sonic | Not needed at this scale; stdlib is fast enough; json/v2 available experimentally. |
| Storage | in-memory map + mutex | SQLite / Postgres | Out of scope (no persistence, single process). |
| Font | System TTF (Arial) via `FONT_PATH` | Bundled font in repo | Larger repo; env override keeps flexibility; document fallback. |

## Installation / Module Setup

```bash
# Module already initialized (go.mod: module ocrpdf, go 1.26.5)
# Add direct dependencies once source imports them, then tidy:
go get github.com/pdfcpu/pdfcpu@v0.16.3
go get github.com/fogleman/gg@v1.3.0
go get golang.org/x/image@v0.47.0
# Optional (CGO, build-tagged):
go get github.com/otiai10/gosseract/v2@v2.4.1
go mod tidy
```

```bash
# Default build (no CGO)
go build ./...
go test ./...

# Optional CGO backend
# (Linux) sudo apt install libtesseract-dev libleptonica-dev
go build -tags gosseract ./...
```

## Version / Compatibility Notes

- `go.mod` currently marks all five requires as `// indirect` with an empty `go.sum`; `go mod tidy` with no source removes them. Once implementation imports them, re-run `go mod tidy` so Go reclassifies them as direct.
- pdfcpu v0.16.3 is not marked "stable" on pkg.go.dev (pre-1.0); pin the version and wrap calls behind `internal/pdfio` (API drift risk).
- gosseract v2.4.1 is the latest stable (2023-09-06); no newer release. API naming in older plans (`GetTextBoxes`) is wrong - use `GetBoundingBoxesVerbose()`.
- gg v1.3.0 is unmaintained (last release 2019) but stable API; keep usage inside `internal/edit`.
- x/image v0.47.0 and Go 1.26.5 are very recent; contributors on older Go cannot build. Consider lowering the `go` directive to the minimum actually required (likely 1.22+) after verifying APIs.
- Tesseract CLI TSV has 12 columns and level-5 rows; parse with `encoding/csv` (tab delimiter), skip rows where level != 5 or conf < 0.

## Sources

- pkg.go.dev: pdfcpu v0.16.3, gosseract v2.4.1, gg v1.3.0, golang.org/x/image v0.47.0, golang/freetype (verified 2026-10-27)
- Installed toolchain: `go version go1.26.5 windows/amd64`; Tesseract 5.5.0.20241111 at `C:\Program Files\Tesseract-OCR\tesseract.exe`
- Project: `.planning/PROJECT.md`, `.planning/codebase/{STACK,ARCHITECTURE,INTEGRATIONS,CONCERNS}.md`
