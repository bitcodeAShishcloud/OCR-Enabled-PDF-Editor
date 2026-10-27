# Phase 5: Edit, Rebuild & End-to-End Verification - Context

**Gathered:** 2026-10-28
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the edit pipeline (erase + render with font-fit and color sampling), the rebuild/download endpoint, and the synthetic end-to-end verification test. After this phase, the project is complete and verified.

</domain>

<decisions>
## Implementation Decisions

### Erase
- **D-01:** Sample background color from the pixel ring around the bbox; fill the padded bbox. Reversible.

### Render
- **D-02:** Auto-fit font size to bbox width via binary search. Reversible.
- **D-03:** Sample text color from old glyphs' darkest pixels. Reversible.
- **D-04:** Baseline-aligned to bbox bottom using font metrics. Reversible.
- **D-05:** Load TTF from `FONT_PATH` (default Arial). Reversible.

### Edit Endpoint
- **D-06:** `POST /api/documents/{id}/edits` accepts page, oldText or wordIds, newText, optional font/size/color. Returns edited preview. Reversible.

### Rebuild Endpoint
- **D-07:** `GET /api/documents/{id}/result.pdf` rebuilds from current edited images and streams. Reversible.

### E2E Test
- **D-08:** Synthetic: gg draws a name onto an image → pdfcpu imports to PDF → upload → OCR → edit → rebuild → re-OCR verifies new text. Reversible.

### the agent's Discretion
- Preview response shape (image bytes vs URL).
- Color sampling heuristic details.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/PROJECT.md`
- `.planning/REQUIREMENTS.md` — EDIT-01..06, API-03, API-04, TEST-02, TEST-03
- `.planning/ROADMAP.md` — Phase 5
- `.planning/research/PITFALLS.md` — coordinate/baseline/color sampling

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/pdfio`, `internal/ocr`, `internal/api` all complete.
- `fogleman/gg`, `golang.org/x/image`, freetype in go.mod.

### Established Patterns
- stdlib net/http, mutex store, pdfcpu I/O.

### Integration Points
- Final phase; produces SKELETON completion.

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
*Phase: 05-edit-rebuild-end-to-end-verification*
*Context gathered: 2026-10-28*