# Phase 3: OCR Engine & TSV Parsing - Context

**Gathered:** 2026-10-27
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the `internal/ocr` package fully: CLI backend parsing Tesseract TSV into word-level text/conf/bbox, gosseract backend behind build tag, unit tests for TSV parsing. No HTTP, no editing.

</domain>

<decisions>
## Implementation Decisions

### TSV Parsing
- **D-01:** Use `encoding/csv` with tab delimiter; filter `level == 5`; ignore `-1` confidence; handle header/blank lines and variable field counts. Reversible.
- **D-02:** Word bbox in image pixel coordinates (x0,y0,x1,y1). Reversible.

### CLI Backend
- **D-03:** Invoke tesseract binary via `os/exec`; `TESSERACT_PATH` override with Windows default; set `TESSDATA_PREFIX` if unset. Reversible.
- **D-04:** Default backend. Reversible.

### gosseract Backend
- **D-05:** Behind `//go:build gosseract`; use `GetBoundingBoxesVerbose`; stub returns clear error when not built. Reversible.

### the agent's Discretion
- Temp file handling for tesseract input.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/PROJECT.md`
- `.planning/REQUIREMENTS.md` — OCR-02..05, TEST-01
- `.planning/ROADMAP.md` — Phase 3
- `.planning/research/PITFALLS.md` — TSV parsing edge cases
- `.planning/phases/01-project-bootstrap-configuration/01-RESEARCH.md`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Phase 1 created `internal/ocr/ocr.go` (Engine, Word, Config, NewEngine) and stubs.

### Established Patterns
- Packages under `internal/`. Tests in same package.

### Integration Points
- Consumed by `internal/api` in Phase 4.

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
*Phase: 03-ocr-engine-tsv-parsing*
*Context gathered: 2026-10-27*