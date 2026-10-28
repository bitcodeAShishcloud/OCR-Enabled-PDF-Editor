# Phase 6: React Web UI - Research

**Researched:** 2026-10-28
**Domain:** React 18 + TypeScript + Vite + Tailwind SPA consuming the Go REST API, served by Go
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** React 18 + TypeScript + Vite. Reversible.
- **D-02:** Tailwind CSS for styling. Reversible.
- **D-03:** React Router for client-side routing. Reversible.
- **D-04:** Frontend lives in `web/` directory at repo root. Reversible.
- **D-05:** Go backend serves built `web/dist` as static files via `http.FileServer` with SPA fallback. Reversible.
- **D-06:** Use fetch (no axios) for API calls; proxy in Vite dev to Go backend. Reversible.
- **D-07:** API base path `/api`; Vite dev proxy forwards `/api` to `http://localhost:8080`. Reversible.
- **D-08:** Pages: Upload (drag/drop), Document view (page preview + word overlays + edit dialog), Download result. Reversible.
- **D-09:** Word overlay: clickable bbox rectangles over the page PNG; clicking opens edit dialog with oldText pre-filled. Reversible.
- **D-10:** Edit dialog: newText input, optional font/size/color, submit → refresh preview. Reversible.
- **D-11:** `npm run build` produces `web/dist`; Go embeds via `//go:embed` or serves from disk. Use disk-based serving with `WEB_DIST` env override for dev. Reversible.

### the agent's Discretion
- Component decomposition, state management (useState/useReducer).
- Exact styling details.

### Deferred Ideas (OUT OF SCOPE)
- WebSocket for real-time progress.
- Mobile responsiveness polish.
</user_constraints>

## Summary

Phase 6 adds a browser UI on top of the already-complete Go REST API. No new backend endpoints are required: the frontend consumes `POST /api/documents`, `GET /api/documents/{id}/pages/{n}`, `POST /api/documents/{id}/edits`, `GET /api/documents/{id}/result.pdf`, and `DELETE /api/documents/{id}` (registered in `cmd/server/main.go:112-116`). The work is (a) scaffold a Vite + React 18 + TS + Tailwind project under `web/`, (b) implement Upload / Document (page preview + clickable word overlays + edit dialog) / Download flows with fetch, (c) wire Vite dev proxy `/api` → `http://localhost:8080`, and (d) teach the Go server to serve `web/dist` with SPA fallback and a `WEB_DIST` env override.

Recommended approach: disk-based static serving (per D-11) with `WEB_DIST` override, because `//go:embed` of `web/dist` would fail `go build` whenever the frontend has not been built (CI, fresh clone). Optionally expose embedding behind a build tag for a single-binary production build. Standard stack: Vite 8.3.5, React 18.3.1, TypeScript 7.0.2 (5.x TS types), Tailwind 4.3.3 via `@tailwindcss/vite`, `@vitejs/plugin-react` 6.1.2, React Router 7.18.5. Node 22.17.0 and npm 11.8.0 are available in the dev environment, so the frontend can be built and verified locally.

**Primary recommendation:** Scaffold with `npm create vite@latest web -- --template react-ts`, add Tailwind v4 via the Vite plugin, implement a small fetch client against `/api`, and serve `web/dist` from Go with an SPA fallback handler that rewrites unknown paths to `index.html`. Do not embed by default.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Upload drag/drop + multipart POST | Browser / Client | API / Backend | UI collects file; backend owns validation/extraction/OCR (`handlers.go:upload`) |
| Page PNG preview rendering | Browser / Client | CDN / Static | `<img>` fetches `GET /api/documents/{id}/pages/{n}` (backend streams PNG) |
| Word bbox overlay + click target | Browser / Client | — | Pure presentation; bboxes come from upload response (`wordDTO`, `handlers.go:46-50`) |
| Edit dialog + submit edit | Browser / Client | API / Backend | UI collects newText/options; backend erases+renders (`handlers.go:edit`, `editRequest`, `handlers.go:267-275`) |
| Refresh preview after edit | Browser / Client | — | Re-fetch page PNG with cache-bust (edit response also returns PNG bytes) |
| Download rebuilt PDF | Browser / Client | API / Backend | Anchor to `GET /api/documents/{id}/result.pdf`; backend rebuilds (`handlers.go:result`) |
| Static asset serving (JS/CSS/HTML) | API / Backend (Go) | CDN / Static | Go `http.FileServer` over `web/dist` with SPA fallback (D-05, D-11) |
| Dev-time proxy `/api` → Go | Frontend dev server (Vite) | — | Vite `server.proxy` (D-07) keeps a single origin in dev |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 18.3.1 | UI components, state | Locked by D-01; latest 18.x line |
| react-dom | 18.3.1 | DOM rendering | Paired with React |
| TypeScript | 5.x (use 5.6+) | Type safety for API DTOs | Locked by D-01; `@types/react` 18.3.x |
| Vite | 8.3.5 | Dev server + bundler | Locked by D-01; `@vitejs/plugin-react` 6.1.2 targets Vite ^8 |
| Tailwind CSS | 4.3.3 | Utility styling | Locked by D-02; v4 uses Vite plugin + CSS `@import` |
| @tailwindcss/vite | 4.3.3 | Tailwind Vite integration | Official v4 path; no PostCSS config needed |
| @vitejs/plugin-react | 6.1.2 | React Fast Refresh + JSX | Official React plugin for Vite |
| React Router | 7.18.5 | Client-side routing | Locked by D-03; `react-router-dom` provides DOM bindings |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| @types/react | 18.3.31 | React type defs | Dev only |
| @types/react-dom | 18.3.x | ReactDOM type defs | Dev only |
| Vitest (optional) | latest | Component unit tests | Only if adding frontend tests; not required by v1 reqs |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Tailwind v4 Vite plugin | PostCSS + tailwind.config.js (v3 style) | v3 LTS still available (3.4.19) but v4 is current; D-02 just says Tailwind, v4 is the current line |
| React Router | TanStack Router | D-03 locks React Router |
| fetch | axios | D-06 locks fetch; no extra dependency |

**Installation:**
```bash
cd web
npm install react@18.3.1 react-dom@18.3.1 react-router-dom@7.18.5
npm install -D vite@8.3.5 @vitejs/plugin-react@6.1.2 typescript@5.6 @types/react@18.3.31 @types/react-dom@18.3 tailwindcss@4.3.3 @tailwindcss/vite@4.3.3
```

**Version verification:** Versions above confirmed via `npm view <pkg> version` on 2026-10-28 against the npm registry. `[VERIFIED: npm registry]` for each listed version. Note: React 19.3.1 is the absolute latest, but D-01 locks React 18, so 18.3.1 is pinned.

## Package Legitimacy Audit

Run via `gsd-tools query package-legitimacy check --ecosystem npm` on 2026-10-28.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| react | npm | published 2026-10-14 (latest line) | 165M/wk | github.com/react/react | [SUS] (too-new heuristic) | Approved — official source repo confirmed; pin to 18.3.1 |
| react-dom | npm | published 2026-10-14 | 155M/wk | github.com/react/react | [SUS] (too-new) | Approved |
| vite | npm | published 2026-10-21 | 171M/wk | github.com/vitejs/vite | [SUS] (too-new) | Approved |
| typescript | npm | published 2026-07-08 | 267M/wk | github.com/microsoft/TypeScript | [OK] | Approved |
| tailwindcss | npm | published 2026-07-16 | 115M/wk | github.com/tailwindlabs/tailwindcss | [OK] | Approved |
| @vitejs/plugin-react | npm | published 2026-10-05 | 82M/wk | github.com/vitejs/vite-plugin-react | [SUS] (too-new) | Approved |
| @tailwindcss/vite | npm | published 2026-07-16 | 39M/wk | github.com/tailwindlabs/tailwindcss | [OK] | Approved |
| react-router-dom | npm | published 2026-10-27 | 37M/wk | github.com/remix-run/react-router | [SUS] (too-new) | Approved |
| react-router | npm | published 2026-10-27 | 49M/wk | github.com/remix-run/react-router | [SUS] (too-new) | Approved |
| @types/react | npm | published 2026-09-09 | 149M/wk | github.com/DefinitelyTyped/DefinitelyTyped | [OK] | Approved |
| @types/react-dom | npm | published 2026-09-09 | 127M/wk | github.com/DefinitelyTyped/DefinitelyTyped | [OK] | Approved |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** react, react-dom, vite, @vitejs/plugin-react, react-router-dom, react-router — all flagged only because of the seam's "too-new" heuristic (published within ~2–4 weeks of the current date). Each resolves to an official source repository with tens of millions of weekly downloads and `postinstall: null`. No `postinstall` scripts present. Treat as approved; the "too-new" flag is a false positive for these well-known packages. The planner may still add a `checkpoint:human-verify` before the first `npm install` if desired, but it is not required to block execution.

## Architecture Patterns

### System Architecture Diagram

```text
Browser (React SPA)
   |
   |-- dev: Vite dev server :5173 --proxy /api--> Go :8080
   |-- prod: Go serves web/dist (static + SPA fallback)
   |
   v
Go http.ServeMux (cmd/server/main.go)
   |-- GET  /healthz
   |-- POST /api/documents          -> Upload (multipart) -> docId + pages/words
   |-- GET  /api/documents/{id}/pages/{n}  -> PNG
   |-- POST /api/documents/{id}/edits      -> edited PNG preview
   |-- GET  /api/documents/{id}/result.pdf -> rebuilt PDF
   |-- DELETE /api/documents/{id}         -> cleanup
   |-- GET  /*  -> SPA handler (web/dist, fallback index.html)
   |
   v
In-memory Store (mutex) + temp dirs + Tesseract/pdfcpu/gg
```

### Recommended Project Structure
```
web/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── tsconfig.app.json
├── tsconfig.node.json
├── index.html
├── public/                 # copied verbatim to dist
└── src/
    ├── main.tsx            # createRoot + BrowserRouter
    ├── App.tsx             # Routes
    ├── index.css           # @import "tailwindcss";
    ├── api/
    │   ├── client.ts       # fetch wrapper, base /api
    │   └── types.ts        # DTO interfaces mirroring handlers.go
    ├── components/
    │   ├── UploadDropzone.tsx
    │   ├── PagePreview.tsx # img + absolutely-positioned bbox overlays
    │   ├── WordOverlay.tsx
    │   └── EditDialog.tsx
    └── pages/
        ├── UploadPage.tsx
        └── DocumentPage.tsx
```

### Pattern 1: Vite dev proxy to Go backend
**What:** Forward `/api` (and page image/PDF responses under it) to the Go server so the SPA and API share one origin in dev.
**When to use:** Always in dev (`npm run dev`).
**Example:**
```ts
// web/vite.config.ts
// Source: https://vite.dev/config/server-options#server-proxy
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

### Pattern 2: Tailwind v4 via Vite plugin
**What:** Import Tailwind in CSS with `@import "tailwindcss";` and register the plugin; no `tailwind.config.js` required for v4 defaults.
**Example:**
```css
/* web/src/index.css */
@import "tailwindcss";
```
```ts
// vite.config.ts plugins include tailwindcss()
```
Source: https://tailwindcss.com/docs/installation/using-vite

### Pattern 3: Go disk-based static serving with SPA fallback
**What:** Serve files from `WEB_DIST` (default `web/dist`) on disk; if a requested path is not an existing file, serve `index.html` so React Router handles the route. Register after the more specific `/api/...` and `/healthz` routes so they take precedence.
**When to use:** Default (D-11). Avoids embedding unbuilt assets.
**Example:**
```go
// cmd/server/main.go (addition)
func spaHandler(distDir string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" {
            http.NotFound(w, r)
            return
        }
        rel := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
        if rel == "." { rel = "index.html" }
        full := filepath.Join(distDir, rel)
        if info, err := os.Stat(full); err == nil && !info.IsDir() {
            http.ServeFile(w, r, full)
            return
        }
        http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
    })
}
```
Registration (after API routes, in `main()`):
```go
distDir := os.Getenv("WEB_DIST")
if distDir == "" { distDir = "web/dist" }
if info, err := os.Stat(distDir); err == nil && info.IsDir() {
    mux.Handle("GET /", spaHandler(distDir))
}
```
Note: Go 1.22 ServeMux matches `GET /` as a catch-all only when no more specific pattern matches, so `/api/...` and `/healthz` keep precedence. `[CITED: pkg.go.dev/net/http]`

### Pattern 4: Optional embed for single-binary prod builds
**What:** Keep `//go:embed` behind a build tag (e.g. `//go:build embedweb`) so default builds never require `web/dist`. This is optional; disk serving is the default per D-11.
```go
//go:build embedweb
// +build embedweb

package webassets

import "embed"

//go:embed all:dist
var Dist embed.FS
```
Source: https://pkg.go.dev/embed (FS implements fs.FS; usable with http.FileServer(http.FS(...))). `[CITED: pkg.go.dev/embed]`

### Pattern 5: Word overlay scaling
**What:** BBoxes from the API are in original image pixel coordinates (`bboxDTO`, `handlers.go:39-44`). The rendered `<img>` may be scaled by CSS. Compute `scale = displayedWidth / naturalWidth` and position overlay `<div>`s absolutely with `left/top/width/height` multiplied by scale.
**When to use:** PagePreview component.

### Anti-Patterns to Avoid
- **Embedding `web/dist` unconditionally:** `//go:embed web/dist` breaks `go build` on a fresh clone or in CI where the frontend hasn't been built. Use disk serving by default; gate embed behind a build tag.
- **Using `http.FileServer` directly without SPA fallback:** Deep links to client routes (e.g. `/documents/:id`) return 404 because no file exists. Always fall back to `index.html`.
- **Forgetting cache-bust on page images:** After an edit, `GET /api/documents/{id}/pages/{n}` returns the updated PNG but the browser may serve a cached copy. Append `?v=<timestamp>` to the src, or consume the edit response's PNG bytes directly.
- **Rendering OCR text via `dangerouslySetInnerHTML`:** React escapes by default; keep it that way to avoid XSS from untrusted OCR output.
- **Hardcoding `http://localhost:8080` in the client:** Use relative `/api` paths so the same build works in dev (proxied) and prod (same origin).
- **Registering the SPA catch-all before API routes:** Go 1.22 ServeMux specificity handles this, but explicit ordering (API first) is safer and clearer.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Drag/drop file upload | Custom DnD event handlers | Native `<input type="file">` + drag events or a tiny helper | Edge cases (multiple files, directories); keep it simple |
| HTTP client | Custom XHR wrapper | `fetch` (locked D-06) | Standard, no dependency |
| CSS framework | Hand-written CSS | Tailwind v4 (D-02) | Consistency |
| Client routing | Hash-based manual router | React Router (D-03) | Nested routes, params |
| Static file serving | Custom file reader | `http.FileServer` / `http.ServeFile` | Range requests, content types, caching headers |
| SPA fallback | Custom 404 handler guessing | Stat-then-fallback (Pattern 3) | Correct and minimal |

**Key insight:** The backend already does all heavy lifting; the frontend is a thin presentation layer over fetch calls. Avoid introducing state libraries (Redux, etc.) — `useState`/`useReducer` suffice for this scope (per agent's discretion).

## Common Pitfalls

### Pitfall 1: Stale page image after edit
**What goes wrong:** Edit succeeds but the preview still shows the old page.
**Why it happens:** Browser caches `GET /api/documents/{id}/pages/{n}`; the URL is unchanged.
**How to avoid:** Append a cache-busting query param (`?v=${Date.now()}`) after a successful edit, or use the PNG bytes returned by the edit endpoint directly as the preview source (createObjectURL).
**Warning signs:** Edit dialog closes but image unchanged until hard refresh.

### Pitfall 2: Overlay misalignment on resize
**What goes wrong:** Word rectangles drift from glyphs when the image is scaled or window resized.
**Why it happens:** BBoxes are in source-image pixels but overlays are in CSS pixels; scale not recomputed.
**How to avoid:** Recompute scale in a `ResizeObserver` on the image container; store `naturalWidth`/`naturalHeight`.

### Pitfall 3: Multipart field name mismatch
**What goes wrong:** Upload returns 400 "missing 'file' part".
**Why it happens:** Backend reads `r.FormFile("file")` (`handlers.go:103`). The frontend must append the file under the field name `file`.
**How to avoid:** `formData.append('file', file)`.

### Pitfall 4: Edit target resolution
**What goes wrong:** Editing by `oldText` replaces the first matching word only (`handlers.go:325-330` breaks after first match). For a clicked word, prefer `wordIds: [index]` to target the exact occurrence.
**How to avoid:** Track the word's index in `page.words` and send `wordIds`.

### Pitfall 5: Go ServeMux pattern syntax
**What goes wrong:** Registering `mux.Handle("/", h)` (Go <1.22 style) vs `mux.HandleFunc("GET /", ...)`. On Go 1.26, use method-prefixed patterns. API routes already use `"POST /api/documents"` etc. (`main.go:112-116`).
**How to avoid:** Use `"GET /"` for the catch-all and register API routes first.

### Pitfall 6: WEB_DIST relative path
**What goes wrong:** Server started from a different working directory cannot find `web/dist`.
**How to avoid:** Allow `WEB_DIST` to be an absolute path; document that the default `web/dist` is relative to the process cwd. Optionally resolve relative to the executable location.

### Pitfall 7: Tailwind v4 source scanning
**What goes wrong:** Dynamically constructed class names (e.g. `text-${color}`) are purged.
**How to avoid:** Use complete class names; configure `@source` if needed. For this app, static class names suffice.

## Code Examples

### TypeScript DTOs mirroring the Go API
```ts
// web/src/api/types.ts
// Source: internal/api/handlers.go:39-60
export interface BBox { x0: number; y0: number; x1: number; y1: number }
export interface Word { text: string; conf: number; bbox: BBox }
export interface Page { page: number; words: Word[] }
export interface UploadResponse { docId: string; pages: Page[] }
export interface EditRequest {
  page: number
  oldText?: string
  wordIds?: number[]
  newText: string
  fontPath?: string
  fontSize?: number
  color?: string
}
```

### Fetch client
```ts
// web/src/api/client.ts
const BASE = '/api'

export async function uploadPdf(file: File): Promise<UploadResponse> {
  const fd = new FormData()
  fd.append('file', file) // field name must be "file" (handlers.go:103)
  const res = await fetch(`${BASE}/documents`, { method: 'POST', body: fd })
  if (!res.ok) throw new Error((await res.json()).error ?? res.statusText)
  return res.json()
}

export async function getPageImage(docId: string, page: number): Promise<string> {
  // returns an object URL for the PNG bytes
  const res = await fetch(`${BASE}/documents/${docId}/pages/${page}`)
  if (!res.ok) throw new Error((await res.json()).error ?? res.statusText)
  const blob = await res.blob()
  return URL.createObjectURL(blob)
}

export async function applyEdit(docId: string, req: EditRequest): Promise<Blob> {
  const res = await fetch(`${BASE}/documents/${docId}/edits`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
  if (!res.ok) throw new Error((await res.json()).error ?? res.statusText)
  return res.blob() // edited preview PNG (handlers.go:429-430)
}

export function resultUrl(docId: string): string {
  return `${BASE}/documents/${docId}/result.pdf`
}
```

### Word overlay component
```tsx
// web/src/components/PagePreview.tsx
import { useEffect, useRef, useState } from 'react'
import type { Page, Word } from '../api/types'

export function PagePreview({ src, page, onWordClick }: {
  src: string; page: Page; onWordClick: (w: Word, idx: number) => void
}) {
  const imgRef = useRef<HTMLImageElement>(null)
  const [scale, setScale] = useState(1)
  useEffect(() => {
    const img = imgRef.current
    if (!img) return
    const update = () => {
      if (img.naturalWidth) setScale(img.clientWidth / img.naturalWidth)
    }
    const ro = new ResizeObserver(update)
    ro.observe(img)
    img.addEventListener('load', update)
    return () => { ro.disconnect(); img.removeEventListener('load', update) }
  }, [src])
  return (
    <div className="relative inline-block">
      <img ref={imgRef} src={src} alt={`page ${page.page}`} className="max-w-full" />
      {page.words.map((w, i) => (
        <button
          key={i}
          onClick={() => onWordClick(w, i)}
          className="absolute border border-blue-500/60 hover:bg-blue-500/20 cursor-pointer"
          style={{
            left: w.bbox.x0 * scale,
            top: w.bbox.y0 * scale,
            width: (w.bbox.x1 - w.bbox.x0) * scale,
            height: (w.bbox.y1 - w.bbox.y0) * scale,
          }}
          aria-label={w.text}
        />
      ))}
    </div>
  )
}
```

### Upload dropzone (minimal)
```tsx
<input type="file" accept="application/pdf" onChange={e => e.target.files?.[0] && onFile(e.target.files[0])} />
```

### Download anchor
```tsx
<a href={resultUrl(docId)} download="result.pdf" className="btn">Download PDF</a>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Tailwind via PostCSS + config file | Tailwind v4 Vite plugin + `@import "tailwindcss"` | v4 (2025) | Simpler setup, no config file needed |
| Vite React plugin (classic) | `@vitejs/plugin-react` 6.x (Oxc-based) | 2026 | Faster transforms; requires Vite ^8 |
| `http.FileServer` with `http.Dir` | Same, plus SPA fallback wrapper | stable | Still the idiomatic Go approach |

**Deprecated/outdated:**
- `tailwind.config.js` content paths: v4 auto-detects sources; manual config optional.
- `create-react-app`: superseded by Vite; do not use.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Component decomposition and state management use only `useState`/`useReducer` | Architecture Patterns | Low — can add a store later without API changes |
| A2 | Word overlay uses `<button>` elements positioned absolutely; no canvas | Code Examples | Low — accessibility better; performance fine for typical page word counts |
| A3 | Edit refresh uses the PNG bytes returned by the edit endpoint (blob URL) rather than re-fetching the page | Common Pitfalls | Low — either approach works; blob URL avoids cache issues |
| A4 | `WEB_DIST` default resolved relative to process cwd | Architecture Patterns | Medium — document absolute path usage; could resolve relative to executable instead |
| A5 | No frontend automated tests required for v1 (nyquist_validation is false) | Environment/Validation | Low — manual UAT sufficient; Vitest optional |

## Open Questions

1. **Should the Go binary embed `web/dist` for production?**
   - What we know: D-11 allows either; disk serving chosen for dev safety.
   - What's unclear: Whether a single-binary deployment is desired.
   - Recommendation: Ship disk-based by default; add the `embedweb` build tag as an opt-in (Pattern 4) only if single-binary is required. Confirm with user.

2. **Port for Vite dev server and Go backend coordination**
   - What we know: Go defaults to 8080 (PORT env, `main.go:45-48`); Vite defaults to 5173.
   - What's unclear: Whether to change Go PORT in dev.
   - Recommendation: Keep Go on 8080, Vite on 5173; proxy handles integration.

3. **Cleanup on page unload**
   - What we know: `DELETE /api/documents/{id}` exists (`main.go:114`).
   - What's unclear: Whether to call it on `beforeunload`.
   - Recommendation: Call it on unmount/unload to avoid temp dir leaks; non-blocking (fire-and-forget).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Vite build/dev | ✓ | 22.17.0 | — |
| npm | Package install | ✓ | 11.8.0 | — |
| Go toolchain | Backend build | ✓ | 1.26.5 (go.mod) | — |
| Tesseract | OCR (backend) | ✓ | 5.5.0 (Windows) | — |
| Go static serving (http.FileServer) | Serving web/dist | ✓ (stdlib) | Go 1.26 | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none.

Note: `web/dist` does not exist yet (no `web/` directory). The first plan must scaffold it; Go serving must guard on directory existence so the backend still boots without the frontend built.

## Security Domain

Security enforcement enabled (config `security_enforcement: true`, ASVS L1).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | No auth in scope (out of scope per REQUIREMENTS.md) |
| V3 Session Management | no | Stateless; no sessions |
| V4 Access Control | no | Single-process, no users |
| V5 Input Validation | yes (backend) | Backend validates PDF magic bytes, size/page caps (`handlers.go:128-181`); frontend validates file type via `accept` |
| V6 Cryptography | no | No crypto in frontend |

### Known Threat Patterns for {stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| XSS via OCR text rendered in DOM | Tampering | React's default escaping; never use `dangerouslySetInnerHTML` |
| Path traversal in `WEB_DIST` serving | Tampering | `filepath.Clean` + `filepath.Join` + `os.Stat` existence check before serving |
| CSRF on edit/upload endpoints | Spoofing | Same-origin in prod; dev proxy keeps single origin; no cookies/auth tokens |
| Clickjacking of overlay UI | Spoofing | Add `X-Frame-Options: DENY` / CSP `frame-ancestors 'none'` (optional hardening) |
| Leaked temp dirs on failed uploads | Info disclosure | Backend cleanup (`os.RemoveAll` on failure, `handlers.go:145-149`); frontend calls DELETE on unload |

## Sources

### Primary (HIGH confidence)
- npm registry (`npm view`) — versions for react, react-dom, vite, typescript, tailwindcss, @vitejs/plugin-react, @tailwindcss/vite, react-router-dom, react-router, @types/react, @types/react-dom [VERIFIED: npm registry]
- https://vite.dev/config/server-options#server-proxy — Vite proxy configuration [CITED: vite.dev]
- https://tailwindcss.com/docs/installation/using-vite — Tailwind v4 Vite setup [CITED: tailwindcss.com]
- https://pkg.go.dev/embed — `//go:embed` directives and `embed.FS` [CITED: pkg.go.dev]
- `cmd/server/main.go:102-116` — route registration and config [VERIFIED: cmd/server/main.go:102-116]
- `internal/api/handlers.go:39-60` — DTO shapes [VERIFIED: internal/api/handlers.go:39-60]
- `internal/api/handlers.go:267-275` — editRequest shape [VERIFIED: internal/api/handlers.go:267-275]
- `internal/api/handlers.go:26-29` — DefaultMaxUploadBytes/MaxPages [VERIFIED: internal/api/handlers.go:26-29]

### Secondary (MEDIUM confidence)
- Go `net/http` ServeMux pattern matching (Go 1.22+) — method-prefixed patterns and catch-all specificity [CITED: pkg.go.dev/net/http]

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified against npm registry on 2026-10-28
- Architecture: HIGH — patterns verified against official Vite/Tailwind/Go docs and existing Go API code
- Pitfalls: MEDIUM — derived from code inspection of handlers.go and common SPA pitfalls

**Research date:** 2026-10-28
**Valid until:** 2026-11-27 (30 days; Vite/Tailwind move fast, re-verify versions before execution)
