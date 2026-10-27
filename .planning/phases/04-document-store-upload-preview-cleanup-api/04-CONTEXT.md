# Phase 4: Document Store & Upload/Preview/Cleanup API - Context

**Gathered:** 2026-10-27
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the HTTP API layer: document store (mutex-protected map), upload endpoint (PDF → extract → OCR), page preview (PNG), cleanup (DELETE). No editing/rebuild yet (Phase 5).

</domain>

<decisions>
## Implementation Decisions

### Store
- **D-01:** In-memory map keyed by server-generated UUID; mutex-protected. Reversible.
- **D-02:** Store holds page image paths + OCR words per page. Reversible.

### Upload
- **D-03:** `POST /api/documents` multipart upload; validate magic bytes; enforce size/page caps. Reversible.
- **D-04:** Returns `{docId, pages:[{page, words:[{text, conf, bbox}]}]}`. Reversible.

### Preview
- **D-05:** `GET /api/documents/{id}/pages/{n}` serves PNG with correct content type. Reversible.

### Cleanup
- **D-06:** `DELETE /api/documents/{id}` removes doc + artifacts; later access 404. Reversible.

### IDs
- **D-07:** Server-generated IDs (UUID v4). Reversible.

### the agent's Discretion
- Cap values (size, pages). Error response shape.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/PROJECT.md`
- `.planning/REQUIREMENTS.md` — API-01, 02, 05, 06, 07, TEST-04
- `.planning/ROADMAP.md` — Phase 4
- `.planning/phases/01-project-bootstrap-configuration/01-RESEARCH.md`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/pdfio` (extract), `internal/ocr` (Engine, Word).
- Phase 1 server skeleton (main.go, healthz).

### Established Patterns
- stdlib net/http ServeMux. Mutex-protected store.

### Integration Points
- Consumed by Phase 5 (edit/rebuild).

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
*Phase: 04-document-store-upload-preview-cleanup-api*
*Context gathered: 2026-10-27*