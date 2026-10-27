---
phase: 02-pdf-i-o-layer
plan: 01
subsystem: pdf-io
tags: [pdfcpu, extraction, rebuild, go]

requires:
  - phase: 01-project-bootstrap
    provides: Go module skeleton and internal package layout

provides:
  - internal/pdfio package with ExtractImages, BuildPDF, NewTempDir
  - Largest-by-pixel-area image selection per page
  - Truncating rebuild with page dimensions from source image

affects: [phase-04-api, edit-layer]

tech-stack:
  added: [github.com/pdfcpu/pdfcpu v0.16.3, golang.org/x/image v0.47.0, github.com/hhrutter/tiff v1.0.7]
  patterns: [pdfcpu file-based extraction, os.Remove before import to avoid append, PageDim from image.DecodeConfig]

key-files:
  created:
    - internal/pdfio/extract.go
    - internal/pdfio/build.go
    - internal/pdfio/temp.go
    - internal/pdfio/extract_test.go
    - internal/pdfio/build_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Broadened pageFileRe qualifier class to [A-Za-z0-9]+ because pdfcpu emits names like Im0"
  - "Single PageDim (first image) applies to all pages per D-05"

patterns-established:
  - "Wrap all pdfio errors with pdfio: prefix"
  - "Reject empty paths and filepath.Clean inputs"

requirements-completed: [PDF-01, PDF-02, PDF-03, PDF-04]

actuals:
  tokens: 4290
  tasks: 3
  commits: 2
  plan_head_before: a5feb27
  plan_head_after: 6fb9a91

coverage:
  - id: D1
    description: "ExtractImages returns one image path per page selecting largest by pixel area"
    requirement: PDF-01
    verification:
      - kind: unit
        ref: "internal/pdfio/extract_test.go#TestLargestImagePrefersBiggerArea"
        status: pass
      - kind: integration
        ref: "internal/pdfio/extract_test.go#TestExtractReturnsOnePer"
        status: pass
    human_judgment: false
  - id: D2
    description: "BuildPDF produces one page per image with matching dimensions and no append"
    requirement: PDF-02
    verification:
      - kind: integration
        ref: "internal/pdfio/build_test.go#TestBuildPageCount"
        status: pass
      - kind: integration
        ref: "internal/pdfio/build_test.go#TestBuildPageDim"
        status: pass
      - kind: integration
        ref: "internal/pdfio/build_test.go#TestBuildNoAppend"
        status: pass
    human_judgment: false
  - id: D3
    description: "NewTempDir creates and cleans scratch directory"
    requirement: PDF-04
    verification:
      - kind: unit
        ref: "internal/pdfio/build_test.go#TestNewTempDirCreatesAndCleans"
        status: pass
    human_judgment: false

metrics:
  duration: 25min
  completed: 2026-10-27
status: complete
---

# Phase 2 Plan 01: PDF I/O Layer Summary

**pdfcpu-backed internal/pdfio package extracting largest embedded raster per page and rebuilding PDFs with source-image dimensions, with truncation on retry**

## Performance

- **Duration:** 25 min
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments

- Implemented `ExtractImages` with per-page largest-by-pixel-area selection and missing-page errors.
- Implemented `BuildPDF` removing existing output first and setting `PageDim` from source image.
- Added `NewTempDir` helper for per-document scratch space.
- Added tests covering selection, tie-break, round trip, page count, no-append, dimensions, empty rejection, temp dir.

## Task Commits

1. **Task 1: Implement pdfio package** - `1725f4a` (feat)
2. **Task 2/3: Tests** - `6fb9a91` (test)

## Files Created/Modified

- `internal/pdfio/extract.go` - extraction + largestImage
- `internal/pdfio/build.go` - rebuild with truncation + PageDim
- `internal/pdfio/temp.go` - temp dir helper
- `internal/pdfio/extract_test.go` - extraction tests
- `internal/pdfio/build_test.go` - rebuild tests
- `go.mod`, `go.sum` - pdfcpu and transitive deps

## Decisions Made

- Broadened `pageFileRe` qualifier from `[A-Za-z]+` to `[A-Za-z0-9]+` to match pdfcpu's `Im0`-style resource names.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Regex qualifier class too narrow for pdfcpu output**
- **Found during:** Task 2 (extraction round-trip test)
- **Issue:** Plan specified `_(\\d+)_([A-Za-z]+)\\.`, but pdfcpu emits names like `in_1_Im0.png` where the qualifier contains digits, so no files matched and extraction failed with "no embedded image for page 1".
- **Fix:** Changed qualifier to `[A-Za-z0-9]+`.
- **Files modified:** internal/pdfio/extract.go
- **Commit:** 1725f4a

**Total deviations:** 1 auto-fixed
**Impact on plan:** Necessary for correctness; no scope creep.

## Issues Encountered

- InsertPagesFile used to simulate a vector-only page for the missing-image test; works.

## Next Phase Readiness

- pdfio package ready for Phase 4 consumption.

---
*Phase: 02-pdf-i-o-layer*
*Completed: 2026-10-27*