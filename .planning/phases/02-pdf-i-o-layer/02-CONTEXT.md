# Phase 2: PDF I/O Layer - Context

**Gathered:** 2026-10-27
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the `internal/pdfio` package: extract the largest embedded raster image per page from a scanned PDF (`extract.go`), and rebuild a PDF from page images (`build.go`). No OCR, no HTTP, no editing. Deliverable is a tested, standalone pdfio package.

</domain>

<decisions>
## Implementation Decisions

### Extraction
- **D-01:** Use `pdfcpu` `api.ExtractImagesFile`; enumerate output dir to find all images per page, pick largest by pixel area. Reversible.
- **D-02:** Return `[]string` of image paths (one per page) in page order. Reversible.
- **D-03:** Temp directory per document; caller cleans up. Reversible.

### Rebuild
- **D-04:** Remove existing output file before calling `ImportImagesFile` (no append). Reversible.
- **D-05:** Set `ImportConfig.PageDim` from source image dimensions (no A4 stretch). Reversible.
- **D-06:** Use `api.DefaultImportConfig()` as base. Reversible.

### Errors
- **D-07:** Surface pdfcpu errors wrapped with context. Reversible.

### the agent's Discretion
- Exact temp dir naming and cleanup helpers.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/PROJECT.md`
- `.planning/REQUIREMENTS.md` — PDF-01..04
- `.planning/ROADMAP.md` — Phase 2
- `.planning/research/PITFALLS.md` — pdfcpu multi-image, append, A4 stretch
- `.planning/codebase/ARCHITECTURE.md`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `go.mod` already has pdfcpu v0.16.3.
- Phase 1 created `internal/ocr/` package pattern to follow.

### Established Patterns
- Packages under `internal/`. No cross-imports between leaf packages.

### Integration Points
- `internal/pdfio` consumed by `internal/api` in Phase 4.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---
*Phase: 02-pdf-i-o-layer*
*Context gathered: 2026-10-27*