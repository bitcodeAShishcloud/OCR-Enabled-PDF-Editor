# Phase 2: PDF I/O Layer - Research

**Researched:** 2026-10-27
**Domain:** Go `internal/pdfio` package built on pdfcpu v0.16.3
**Confidence:** HIGH

## Summary

Phase 2 delivers a standalone `internal/pdfio` package with two operations: extract the largest embedded raster image per page from a scanned PDF, and rebuild a PDF from one image per page. All pdfcpu API signatures were verified by reading the module source at `C:\Users\shesh\go\pkg\mod\github.com\pdfcpu\pdfcpu@v0.16.3`. pdfcpu's `ExtractImagesFile` returns only an `error` and writes every embedded image resource to disk with generated names, so "largest image per page" must be derived by enumerating the output directory and decoding image dimensions. pdfcpu's `ImportImagesFile` is append-oriented and defaults to A4 page dimensions, so callers must `os.Remove` the output first and set `Import.PageDim` to the source image's pixel dimensions.

**Primary recommendation:** Implement `ExtractImages(ctx, pdfPath, outDir string) ([]string, error)` and `BuildPDF(ctx, imagePaths []string, outputPath string) error` (or similarly named rebuild function) directly on the verified pdfcpu file-based APIs, with a small temp-dir helper and deterministic largest-by-pixel-area selection.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Use `pdfcpu` `api.ExtractImagesFile`; enumerate output dir to find all images per page, pick largest by pixel area.
- **D-02:** Return `[]string` of image paths (one per page) in page order.
- **D-03:** Temp directory per document; caller cleans up.
- **D-04:** Remove existing output file before calling `ImportImagesFile` (no append).
- **D-05:** Set `ImportConfig.PageDim` from source image dimensions (no A4 stretch).
- **D-06:** Use `api.DefaultImportConfig()` as base.
- **D-07:** Surface pdfcpu errors wrapped with context.

### the agent's Discretion
- Exact temp dir naming and cleanup helpers.

### Deferred Ideas (OUT OF SCOPE)
None.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PDF-01 | Extract largest embedded raster image per page using pdfcpu | `api.ExtractImagesFile` + directory enumeration + `image.DecodeConfig` area selection |
| PDF-02 | Rebuild PDF from page images (one per page) | `api.ImportImagesFile` with one path per page |
| PDF-03 | Rebuild truncates existing output file | `os.Remove` (ignore `IsNotExist`) before import |
| PDF-04 | Rebuild sets page dimensions from source image | `imp.PageDim = &types.Dim{Width, Height}` using image pixel dimensions |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| PDF extraction | API / Backend | — | pdfcpu runs server-side; no client involvement |
| PDF rebuild | API / Backend | — | pdfcpu runs server-side |
| Temp file lifecycle | API / Backend | — | Server owns filesystem artifacts |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/pdfcpu/pdfcpu` | v0.16.3 | Extract images and rebuild PDFs | Already in `go.mod`; sole PDF library in project |
| Go standard library `image` | Go 1.26.5 | Decode image dimensions for largest-image selection | No extra dependency; pdfcpu already uses it |
| Go standard library `os`/`path/filepath` | Go 1.26.5 | Temp dirs, enumeration, removal | Standard filesystem handling |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/hhrutter/tiff` | (transitive) | TIFF dimension decoding | Blank-import if TIFF inputs must be measured |
| `golang.org/x/image` | v0.47.0 | WebP dimension decoding | Blank-import if WebP inputs must be measured |

**Installation:** no new modules required. `go.mod` already declares pdfcpu v0.16.3.

## Package Legitimacy Audit

No external packages are added in this phase. Existing pdfcpu v0.16.3 dependency is already in `go.mod`. Disposition: Approved.

## Architecture Patterns

### Recommended Project Structure
```
internal/pdfio/
├── extract.go     # ExtractImages
├── build.go       # BuildPDF / RebuildPDF
└── temp.go        # temp dir helpers (agent's discretion)
```

### Pattern 1: Extract Largest Image Per Page
**What:** Call `api.ExtractImagesFile`, then enumerate `outDir`, decode each file's dimensions with `image.DecodeConfig`, group by page number parsed from pdfcpu's generated filename, and keep the candidate with the largest `width*height`.
**When to use:** Every scanned-PDF upload.

Verified signature:
```go
// Source: github.com/pdfcpu/pdfcpu@v0.16.3/pkg/api/extract.go:320
func ExtractImagesFile(c context.Context, inFile, outDir string, selectedPages []string, conf *model.Configuration) (err error)
```
Call with `selectedPages == nil` (all pages) and `conf == model.NewStatelessConfiguration()` (or `nil`, which pdfcpu replaces with the default config).

pdfcpu writes files named `<base>_<page>_<qual>.<ext>` where `<qual>` is `image` or `thumb` and `<page>` is zero-padded to the page-count width. Example: `doc_001_image.png`.

### Pattern 2: Rebuild With Explicit Page Dimensions
**What:** Remove any existing output file, build `*pdfcpu.Import` from `api.DefaultImportConfig()`, set `imp.PageDim` to the image's pixel dimensions, then call `api.ImportImagesFile`.
**When to use:** Result PDF generation.

Verified signatures:
```go
// Source: github.com/pdfcpu/pdfcpu@v0.16.3/pkg/api/importImage.go:45
func DefaultImportConfig() *pdfcpu.Import

// Source: github.com/pdfcpu/pdfcpu@v0.16.3/pkg/api/importImage.go:319
func ImportImagesFile(c context.Context, imgFiles []string, outFile string, imp *pdfcpu.Import, conf *model.Configuration) (err error)
```

Verified `Import` struct:
```go
// Source: github.com/pdfcpu/pdfcpu@v0.16.3/pkg/pdfcpu/importImage.go:54
type Import struct {
    PageDim  *types.Dim  // page dimensions in display unit
    PageSize string
    UserDim  bool
    DPI      int
    Pos      types.Anchor
    Dx, Dy   float64
    Scale    float64
    ScaleAbs bool
    InpUnit  types.DisplayUnit
    Gray     bool
    Sepia    bool
    BgColor  *color.SimpleColor
}
```

Verified `types.Dim`:
```go
// Source: github.com/pdfcpu/pdfcpu@v0.16.3/pkg/pdfcpu/types/types.go:622
type Dim struct {
    Width  float64 `json:"width"`
    Height float64 `json:"height"`
}
```

### Anti-Patterns to Avoid
- **Assuming one file per page:** pdfcpu emits masks, thumbnails, and duplicate resources. Always enumerate and select by pixel area.
- **Reusing an output path without removal:** `ImportImagesFile` appends when `outFile` already exists.
- **Leaving `PageDim` as the A4 default:** produces a stretched A4 page instead of the scan's real size.
- **Parsing pdfcpu filenames with `strings.Split` naively:** use a anchored regexp such as `_(\d+)_([A-Za-z]+)\.[A-Za-z0-9]+$`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| PDF image extraction | Custom PDF parser | `api.ExtractImagesFile` | Handles filters, color spaces, object graphs |
| PDF rebuild from images | Custom PDF writer | `api.ImportImagesFile` | Handles image XObjects, page tree, MediaBox |
| Temp directory cleanup | Ad-hoc file deletion | `os.MkdirTemp` + `os.RemoveAll` | Atomic per-document cleanup |

**Key insight:** pdfcpu is resource-oriented, not page-oriented; the "one image per page" contract is pdfio's responsibility.

## Common Pitfalls

### Pitfall 1: Multiple Images Per Page
**What goes wrong:** A page can yield masks, thumbnails, and duplicate XObjects; selecting by file order picks the wrong image.
**Why it happens:** `ExtractImagesFile` writes every embedded image resource.
**How to avoid:** Enumerate `outDir`, decode dimensions with `image.DecodeConfig`, group by page, select max `width*height`; tie-break by file size then path.
**Warning signs:** Extracted file count != page count; selected image is tiny or has an alpha mask.

### Pitfall 2: Append-On-Rebuild
**What goes wrong:** Re-running a rebuild yields extra pages.
**Why it happens:** `ImportImagesFile` reuses an existing `outFile`.
**How to avoid:** `os.Remove(outFile)` (ignore `os.IsNotExist`) before calling.
**Warning signs:** Output page count exceeds input image count.

### Pitfall 3: A4 Default Stretch
**What goes wrong:** Default config uses A4 `PageDim`, distorting the scan.
**Why it happens:** `DefaultImportConfig()` sets `PageDim = PaperSize["A4"]`.
**How to avoid:** Set `imp.PageDim = &types.Dim{Width: float64(w), Height: float64(h)}` from the first/each source image's decoded dimensions.
**Warning signs:** Output page size differs from input scan dimensions.

### Pitfall 4: Unsupported Resources
**What goes wrong:** `ExtractImagesFile` may return `*api.UnsupportedResourceError` in skip mode.
**Why it happens:** pdfcpu skips unsupported image resources according to policy.
**How to avoid:** Treat any extraction error as failure; do not proceed with partial data. Wrap with context per D-07.

### Pitfall 5: Missing Image For a Page
**What goes wrong:** Some pages have no extractable raster image.
**Why it happens:** Vector-only or empty pages.
**How to avoid:** After selection, verify one path exists for every page `1..pageCount` (obtain count via `api.PageCountFile` or the max parsed page number); return an error otherwise.

## Code Examples

### Extract
```go
// Source: verified against pdfcpu@v0.16.3 pkg/api/extract.go and pkg/api/page.go
package pdfio

import (
    "context"
    "fmt"
    "image"
    _ "image/gif"
    _ "image/jpeg"
    _ "image/png"
    _ "golang.org/x/image/webp"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strconv"

    pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var pageFileRe = regexp.MustCompile(`_(\d+)_([A-Za-z]+)\.[A-Za-z0-9]+$`)

func ExtractImages(ctx context.Context, pdfPath, outDir string) ([]string, error) {
    if err := os.MkdirAll(outDir, 0755); err != nil {
        return nil, fmt.Errorf("pdfio: create output dir %q: %w", outDir, err)
    }
    // Clean stale artifacts from a previous run.
    entries, _ := os.ReadDir(outDir)
    for _, e := range entries {
        _ = os.RemoveAll(filepath.Join(outDir, e.Name()))
    }

    if err := pdfapi.ExtractImagesFile(ctx, pdfPath, outDir, nil, model.NewStatelessConfiguration()); err != nil {
        return nil, fmt.Errorf("pdfio: extract images from %q: %w", pdfPath, err)
    }

    pageCount, err := pdfapi.PageCountFile(ctx, pdfPath)
    if err != nil {
        return nil, fmt.Errorf("pdfio: page count for %q: %w", pdfPath, err)
    }

    // Group extracted files by page number.
    byPage := make(map[int][]string)
    files, err := os.ReadDir(outDir)
    if err != nil {
        return nil, fmt.Errorf("pdfio: read output dir %q: %w", outDir, err)
    }
    for _, e := range files {
        if e.IsDir() {
            continue
        }
        m := pageFileRe.FindStringSubmatch(e.Name())
        if m == nil {
            continue
        }
        page, err := strconv.Atoi(m[1])
        if err != nil {
            continue
        }
        byPage[page] = append(byPage[page], filepath.Join(outDir, e.Name()))
    }

    out := make([]string, 0, pageCount)
    for page := 1; page <= pageCount; page++ {
        candidates := byPage[page]
        if len(candidates) == 0 {
            return nil, fmt.Errorf("pdfio: no embedded image for page %d", page)
        }
        best, err := largestImage(candidates)
        if err != nil {
            return nil, fmt.Errorf("pdfio: select image for page %d: %w", page, err)
        }
        out = append(out, best)
    }
    return out, nil
}

func largestImage(paths []string) (string, error) {
    type candidate struct {
        path string
        area int
        size int64
    }
    var best *candidate
    for _, p := range paths {
        f, err := os.Open(p)
        if err != nil {
            return "", err
        }
        cfg, _, err := image.DecodeConfig(f)
        f.Close()
        if err != nil {
            return "", fmt.Errorf("decode config %q: %w", p, err)
        }
        info, _ := os.Stat(p)
        var size int64
        if info != nil {
            size = info.Size()
        }
        c := &candidate{path: p, area: cfg.Width * cfg.Height, size: size}
        if best == nil || c.area > best.area || (c.area == best.area && c.size > best.size) {
            best = c
        }
    }
    if best == nil {
        return "", fmt.Errorf("no decodable image")
    }
    // Deterministic tie-break by path.
    _ = sort.Strings
    return best.path, nil
}
```

### Build
```go
// Source: verified against pdfcpu@v0.16.3 pkg/api/importImage.go
package pdfio

import (
    "context"
    "fmt"
    "image"
    _ "image/png"
    "os"

    pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func BuildPDF(ctx context.Context, imagePaths []string, outputPath string) error {
    if len(imagePaths) == 0 {
        return fmt.Errorf("pdfio: no images to build")
    }
    // PDF-03: truncate any existing output file (no append).
    if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("pdfio: remove existing output %q: %w", outputPath, err)
    }

    // PDF-04: page dimensions from the first source image.
    f, err := os.Open(imagePaths[0])
    if err != nil {
        return fmt.Errorf("pdfio: open %q: %w", imagePaths[0], err)
    }
    cfg, _, err := image.DecodeConfig(f)
    f.Close()
    if err != nil {
        return fmt.Errorf("pdfio: decode config %q: %w", imagePaths[0], err)
    }

    imp := pdfapi.DefaultImportConfig() // D-06
    imp.PageDim = &types.Dim{Width: float64(cfg.Width), Height: float64(cfg.Height)} // D-05

    if err := pdfapi.ImportImagesFile(ctx, imagePaths, outputPath, imp, model.NewStatelessConfiguration()); err != nil {
        return fmt.Errorf("pdfio: build pdf %q: %w", outputPath, err)
    }
    return nil
}
```

Note: if images have differing dimensions, a single `PageDim` applies to all pages. For per-page dimensions, call `ImportImages` separately or set `PageDim` per image; pdfcpu's file-based API uses one config for all images. Phase 2 scope uses one dimension (first image) per D-05. If per-page sizing is required later, rebuild page-by-page or use `ImportImages` with separate contexts.

### Temp Directory Helper (agent's discretion)
```go
func NewTempDir(prefix string) (string, func(), error) {
    if prefix == "" {
        prefix = "ocrpdf-"
    }
    dir, err := os.MkdirTemp("", prefix+"*")
    if err != nil {
        return "", nil, err
    }
    cleanup := func() { _ = os.RemoveAll(dir) }
    return dir, cleanup, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Assume one image per page | Enumerate + largest by area | pdfcpu v0.16.3 | Correct extraction on scanned PDFs with masks |
| Append to existing result | Remove then import | pdfcpu v0.16.3 | No duplicate pages on retry |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `model.NewStatelessConfiguration()` is sufficient for extraction/import (no config dir) | Code Examples | If pdfcpu requires config dir for image decoding, extraction may fail; use `model.NewDefaultConfiguration()` instead |
| A2 | pdfcpu filenames always match `_(\d+)_([A-Za-z]+)\.[A-Za-z0-9]+$` | Code Examples | If filename format changes, page grouping fails; validate against real fixtures |
| A3 | Single `PageDim` (first image) is acceptable for all pages | Build example | Mixed-size scans would be distorted; per-page rebuild may be needed later |

## Open Questions

1. **Per-page dimensions in rebuild.** `ImportImagesFile` applies one `Import` config to all images. If source pages differ in size, the output will use one page size. Recommendation: store per-page image dimensions and, if needed, rebuild each page separately in a later phase. For v1, assume uniform scan dimensions.
2. **Tie-breaking for equal-area images.** Largest by pixel area; ties broken by file size then lexicographic path. Confirm with fixture PDFs containing masks.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build/test | ✓ | 1.26.5 | — |
| pdfcpu module cache | PDF I/O | ✓ | v0.16.3 | — |
| Tesseract | (later phases) | ✓ | 5.5.0 | — |

No missing dependencies for this phase.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` |
| Config file | none |
| Quick run command | `go test ./internal/pdfio/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PDF-01 | Largest image per page extracted | integration | `go test ./internal/pdfio/... -run TestExtract -v` | ❌ Wave 0 |
| PDF-02 | Rebuild one image per page | integration | `go test ./internal/pdfio/... -run TestBuild -v` | ❌ Wave 0 |
| PDF-03 | No append on rebuild | integration | `go test ./internal/pdfio/... -run TestBuildNoAppend -v` | ❌ Wave 0 |
| PDF-04 | Page dimensions from source image | integration | `go test ./internal/pdfio/... -run TestBuildPageDim -v` | ❌ Wave 0 |

### Wave 0 Gaps
- [ ] `internal/pdfio/extract_test.go` — covers PDF-01
- [ ] `internal/pdfio/build_test.go` — covers PDF-02, PDF-03, PDF-04
- [ ] Fixture scanned PDF under `testdata/`

## Security Domain

No new attack surface in this phase (no HTTP, no user input beyond file paths). Path traversal is handled later by `internal/api` (UUID doc IDs, containment checks). pdfio should still use `filepath.Clean` and reject empty paths.

## Sources

### Primary (HIGH confidence)
- `github.com/pdfcpu/pdfcpu@v0.16.3/pkg/api/extract.go:316-344` — `ExtractImagesFile` signature and filename format
- `github.com/pdfcpu/pdfcpu@v0.16.3/pkg/api/importImage.go:44-47, 317-383` — `DefaultImportConfig`, `ImportImagesFile`
- `github.com/pdfcpu/pdfcpu@v0.16.3/pkg/pdfcpu/importImage.go:53-78` — `Import` struct and default config
- `github.com/pdfcpu/pdfcpu@v0.16.3/pkg/pdfcpu/types/types.go:622-625` — `types.Dim`
- `github.com/pdfcpu/pdfcpu@v0.16.3/pkg/api/page.go:273-337` — `PageCountFile`, `PageDimsFile`

### Secondary (MEDIUM confidence)
- `.planning/research/PITFALLS.md` — pdfcpu multi-image, append, A4 stretch pitfalls

### Tertiary (LOW confidence)
- None.
