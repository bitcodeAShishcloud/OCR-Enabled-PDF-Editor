# Phase 6: React Web UI - Context

**Gathered:** 2026-10-28
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase adds a React web frontend that consumes the Go backend REST API. Users can upload a scanned PDF, view OCR'd words overlaid on page images, click a word to replace it, and download the rebuilt PDF. The Go backend serves the built frontend as static files.

</domain>

<decisions>
## Implementation Decisions

### Stack
- **D-01:** React 18 + TypeScript + Vite. Reversible.
- **D-02:** Tailwind CSS for styling. Reversible.
- **D-03:** React Router for client-side routing. Reversible.

### Structure
- **D-04:** Frontend lives in `web/` directory at repo root. Reversible.
- **D-05:** Go backend serves built `web/dist` as static files via `http.FileServer` with SPA fallback. Reversible.

### API Integration
- **D-06:** Use fetch (no axios) for API calls; proxy in Vite dev to Go backend. Reversible.
- **D-07:** API base path `/api`; Vite dev proxy forwards `/api` to `http://localhost:8080`. Reversible.

### Features
- **D-08:** Pages: Upload (drag/drop), Document view (page preview + word overlays + edit dialog), Download result. Reversible.
- **D-09:** Word overlay: clickable bbox rectangles over the page PNG; clicking opens edit dialog with oldText pre-filled. Reversible.
- **D-10:** Edit dialog: newText input, optional font/size/color, submit → refresh preview. Reversible.

### Build
- **D-11:** `npm run build` produces `web/dist`; Go embeds via `//go:embed` or serves from disk. Use disk-based serving with `WEB_DIST` env override for dev. Reversible.

### the agent's Discretion
- Component decomposition, state management (useState/useReducer).
- Exact styling details.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/PROJECT.md`
- `.planning/REQUIREMENTS.md`
- `.planning/ROADMAP.md` — Phase 6
- `.planning/phases/04-document-store-upload-preview-cleanup-api/04-CONTEXT.md` — API shapes
- `.planning/phases/05-edit-rebuild-end-to-end-verification/05-CONTEXT.md` — edit shapes

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Go backend REST API complete and tested.
- ServeMux routing in main.go (add static file handler).

### Established Patterns
- stdlib net/http. Add `/` route serving web/dist with SPA fallback.

### Integration Points
- New `web/` directory. Go serves built assets.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches.

</specifics>

<deferred>
## Deferred Ideas

- WebSocket for real-time progress.
- Mobile responsiveness polish.

</deferred>

---
*Phase: 06-react-web-ui*
*Context gathered: 2026-10-28*