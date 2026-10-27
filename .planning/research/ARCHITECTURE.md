# Architecture Patterns

**Domain:** Go HTTP backend for scanned-PDF OCR text replacement
**Researched:** 2026-10-27

## Recommended Architecture

Layered HTTP service with an interface-based OCR backend and an in-memory, mutex-protected document store. Each edited page is flattened into a raster image; the output PDF is rebuilt from those images. No web framework: standard library `net/http` only, because the surface area is small (5 endpoints) and the heavy work is in external libs (pdfcpu, Tesseract, gg).

```text
+--------------------------------------------------------------+
|                HTTP Client (browser / curl)                  |
+--------------------------------------------------------------+
                          | REST (JSON + PNG + PDF)
                          v
+--------------------------------------------------------------+
|              cmd/server/main.go  (bootstrap)                 |
|  env config: TESSERACT_PATH, TESSDATA_PREFIX, OCR_BACKEND,   |
|  FONT_PATH, PORT  +  net/http server startup                 |
+--------------------------------------------------------------+
                          |
                          v
+--------------------------------------------------------------+
|            internal/api  (HTTP layer)                        |
|  handlers.go: routing, validation, orchestration,            |
|  in-memory document store (sync.Mutex + map[string]Doc)      |
+--------------------------------------------------------------+
        |                  |                  |
        v                  v                  v
+---------------+  +----------------+  +-----------------+
| internal/pdfio|  | internal/ocr   |  | internal/edit   |
| extract.go    |  | ocr.go iface   |  | erase.go        |
| build.go      |  | cli.go (def)   |  | render.go       |
|               |  | gosseract.go   |  |                 |
+---------------+  +----------------+  +-----------------+
        |                  |                  |
        v                  v                  v
+--------------------------------------------------------------+
| External: pdfcpu (extract/build PDF) | Tesseract (CLI/CGO)   |
|           fogleman/gg + golang.org/x/image (render)          |
+--------------------------------------------------------------+
                          |
                          v
+--------------------------------------------------------------+
| Raster store (local filesystem)                              |
|  uploads/<id>.pdf, pages/<id>/<n>.png, edits/<id>/<n>.png,   |
|  result/<id>.pdf                                             |
+--------------------------------------------------------------+
```

### Component Boundaries

| Component | Responsibility | Communicates With | Owns |
|-----------|----------------|-------------------|------|
| `cmd/server/main.go` | Read env, select OCR backend, wire router, start `http.Server`, graceful shutdown | `internal/api`, `internal/ocr` | process lifecycle, config |
| `internal/api` (`handlers.go`) | HTTP handlers, request validation, orchestration across pdfio/ocr/edit, document lifecycle, in-memory store | `internal/pdfio`, `internal/ocr`, `internal/edit` | document map, mutex, route table |
| `internal/pdfio` (`extract.go`, `build.go`) | Bridge PDF <-> per-page raster images; lossless extraction of embedded scanned images; rebuild PDF from images | pdfcpu | page-image files on disk |
| `internal/ocr` (`ocr.go`, `cli.go`, `gosseract.go`) | Recognize words + pixel bboxes from a page image behind `Engine` interface | Tesseract binary or gosseract/v2 | OCR text/confidence |
| `internal/edit` (`erase.go`, `render.go`) | In-place text replacement on a raster page: erase old glyphs, render new text | gg, x/image, font files | edited page images |
| Raster store (filesystem) | Materialized artifacts: uploaded PDF, page PNGs, edited PNGs, result PDF | all of the above (via paths) | bytes on disk |

Dependency direction is strictly one-way: `cmd/server -> internal/api -> internal/{pdfio,ocr,edit} -> external libs`. No package imports `internal/api`; `internal/pdfio`, `internal/ocr`, `internal/edit` do not import each other. This keeps the OCR backend and the edit pipeline independently testable.

### Data Flow

**Upload + OCR path (POST /api/documents):**
1. Client uploads PDF (multipart). `internal/api` handler streams bytes to `uploads/<id>.pdf` (bounded size; reject > configured limit).
2. `internal/pdfio/extract.go` calls pdfcpu `api.ExtractImagesFile` to pull the largest embedded raster image per page into `pages/<id>/<n>.png`.
3. `internal/ocr.Engine.Recognize` runs over each page image. CLI backend: `exec.Command(tesseract, img, "stdout", "tsv")`, parse 12-column TSV level-5 rows. gosseract backend: `GetTextBoxes`.
4. Document record `{id, pageImages, wordsByPage, editedImages}` stored under the mutex in `internal/api`.
5. JSON `{docId, pages:[{page, words:[{text, conf, bbox}]}]}` returned.

**Preview path (GET /api/documents/{id}/pages/{n}):**
- Serve the current page PNG (original or edited) with correct `Content-Type` and `Cache-Control`. Prefer serving from disk via `http.ServeFile`/`http.FileServer` on a scoped subtree; never concatenate user input into a path.

**Edit path (POST /api/documents/{id}/edits):**
1. Validate `{page, oldText|wordIds, newText, font?, size?, color?}`; resolve target words (by text match or explicit ids).
2. `internal/edit/erase.go` samples background color from the pixel ring around the target bbox and fills the padded rect.
3. `internal/edit/render.go` loads TTF (default `C:\Windows\Fonts\arial.ttf`, override `FONT_PATH`), binary-searches font size to fit bbox width, samples text color from old glyphs' darkest pixels, draws replacement text baseline-aligned to bbox bottom.
4. Write edited image to `edits/<id>/<page>.png`; update document record (mark page edited).
5. Return edited preview (PNG or JSON pointing at preview URL).

**Result build path (GET /api/documents/{id}/result.pdf):**
1. For each page, use edited image if present else original.
2. `internal/pdfio/build.go` calls pdfcpu `api.ImportImagesFile` (one image per page) to rebuild the PDF.
3. Stream `result/<id>.pdf` to client with `Content-Disposition: attachment`.

**Cleanup path (DELETE /api/documents/{id}):**
1. Remove in-memory record under mutex.
2. Delete `uploads/<id>.pdf`, `pages/<id>/`, `edits/<id>/`, `result/<id>.pdf` (best-effort; log, don't fail the response on partial FS errors).

### In-Memory Document Store

The store is the single source of truth for a session. Shape (planned):

```go
type Word struct {
    Text       string
    Confidence float64
    X0, Y0, X1, Y1 int  // pixel bbox
}

type Page struct {
    Number     int
    ImagePath  string   // current original page image
    EditedPath string   // non-empty once edited
    Words      []Word
}

type Document struct {
    ID        string
    CreatedAt time.Time
    SourcePDF string
    Pages     []Page
}

type Store struct {
    mu   sync.Mutex
    docs map[string]*Document
}
```

Rules:
- All reads/writes go through `Store` methods that acquire `mu` (or `RLock` if a `sync.RWMutex` is chosen). Never expose the raw map.
- Handlers copy out what they need (e.g., a slice of page paths) while holding the lock, then release before doing I/O. Never hold the mutex across pdfcpu/Tesseract/gg calls.
- IDs are generated with `crypto/rand` (or a UUID lib) — never user-supplied, never sequential, to avoid path traversal and enumeration.
- Store is process-local and non-persistent by design (see anti-patterns). A `Store` interface can be introduced later to swap in disk/DB backing without changing handlers.

## Patterns to Follow

### Pattern 1: Engine Interface (Strategy) for OCR Backends

**What:** A single interface in `internal/ocr/ocr.go` decouples OCR implementation from API and edit layers. Two implementations: CLI (default, no CGO) and gosseract (build-tagged).

**When:** Any place that needs word-level OCR. Selection happens once at startup via `OCR_BACKEND` env var; the chosen instance is injected into handlers.

```go
// internal/ocr/ocr.go
package ocr

type Engine interface {
    Recognize(imagePath string) ([]Word, error)
}

type Word struct {
    Text       string
    Confidence float64
    X0, Y0, X1, Y1 int
}
```

```go
// internal/ocr/cli.go  (default build)
//go:build !gosseract   // or simply no tag; cli always available
package ocr

type CLIEngine struct {
    TesseractPath string
    TesseractPrefix string
}

func (e *CLIEngine) Recognize(imagePath string) ([]Word, error) { ... }
```

```go
// internal/ocr/gosseract.go
//go:build gosseract
package ocr

type GosseractEngine struct { ... }

func (e *GosseractEngine) Recognize(imagePath string) ([]Word, error) { ... }
```

Startup selection (`cmd/server/main.go`):

```go
var eng ocr.Engine
switch os.Getenv("OCR_BACKEND") { // "cli" (default) | "gosseract"
case "gosseract":
    eng = &ocr.GosseractEngine{...}  // only compiles with -tags gosseract
default:
    eng = &ocr.CLIEngine{...}
}
```

Rationale: default build needs no CGO headers (Windows dev friction), CLI works with installed Tesseract 5.5.0 immediately; gosseract is opt-in via `go build -tags gosseract`. Both conform to one interface so handlers and tests are backend-agnostic.

### Pattern 2: Constructor Injection, No Globals

**What:** Handlers receive a `*Store`, an `ocr.Engine`, config (font path, tesseract path), and a base directory for artifacts via a struct. No package-level mutable state.

```go
type Server struct {
    Store   *Store
    OCR     ocr.Engine
    FontPath string
    BaseDir  string
    // ...
}
func (s *Server) Register(mux *http.ServeMux) { ... }
```

**When:** Always. This is what makes the store swappable and the handlers unit-testable (inject a fake Engine, temp dir).

### Pattern 3: Per-Request Context + Timeouts

**What:** Every handler reads `r.Context()`; long-running operations (OCR, rebuild) respect `context.Context` and a server-level timeout. Use `exec.CommandContext` for Tesseract so a cancelled request kills the child process.

**When:** All handler entry points and all `os/exec` calls.

### Pattern 4: Path Safety for Artifact Files

**What:** All on-disk paths are constructed from validated, generated IDs and fixed directory roots. Use `filepath.Join(root, id, name)` then verify the resolved path stays under `root` (`filepath.Rel`/prefix check). Never accept a filename or path segment from the request.

**When:** Upload save, page serve, edit write, result build, cleanup.

### Pattern 5: Atomic-ish File Writes

**What:** Write edited images to a temp file then `os.Rename` into place; delete partial artifacts on error. Avoids serving half-written PNGs.

**When:** Any file that may be concurrently read (page PNGs).

## Anti-Patterns to Avoid

### Anti-Pattern 1: Global Mutable State (package-level store/engine)

**What people do:** `var store = map[string]Document{}` and `var engine ocr.Engine` at package level in `internal/api`.
**Why it's bad:** Hidden coupling, untestable handlers, data races if accessed without lock, impossible to run two instances in one process (tests).
**Instead:** Inject `*Store` and `ocr.Engine` into a `Server` struct; package-level only for constants and pure helpers.

### Anti-Pattern 2: Holding the Mutex Across I/O

**What people do:** `store.mu.Lock(); defer store.mu.Unlock();` then call pdfcpu/Tesseract inside the handler.
**Why it's bad:** Serializes all requests behind one lock; a slow OCR stalls every other client; deadlock risk if a callback re-enters the store.
**Instead:** Acquire lock only to read/copy state or update a record; release before any I/O. Document this as a code-review rule.

### Anti-Pattern 3: Blocking the HTTP Handler with Synchronous Heavy Work (no size/time bounds)

**What people do:** Run OCR on a 200-page PDF in the request goroutine with no timeout or size limit.
**Why it's bad:** Memory blowup, goroutine exhaustion, no cancellation; client hangs.
**Instead:** Enforce upload size limit (`http.MaxBytesReader`), page-count cap, per-request timeout via `context`, and return 413/408 appropriately. For very large docs, consider async job id (out of scope for MVP but note as future).

### Anti-Pattern 4: Default CGO Build / Unguarded gosseract Import

**What people do:** `import "github.com/otiai10/gosseract/v2"` in a file compiled by default.
**Why it's bad:** Breaks `go build` on Windows without MinGW/headers; violates "default build must not require CGO headers" constraint.
**Instead:** gosseract code only in `internal/ocr/gosseract.go` with `//go:build gosseract`; default build uses CLI backend. Verify with `go build ./...` (no tags) and `go vet ./...` in CI.

### Anti-Pattern 5: Path Traversal via User-Supplied IDs/Filenames

**What people do:** `os.ReadFile(filepath.Join(base, r.URL.Path))` or using uploaded filename.
**Why it's bad:** `../` escapes the artifact root; security hole.
**Instead:** Generate IDs server-side; validate `page` is an int in range; construct paths from ID + fixed names only; check containment.

### Anti-Pattern 6: Mutating Shared Page Image Without Copy

**What people do:** Edit in place on the original PNG, losing the pre-edit state and breaking re-edit/revert.
**Why it's bad:** Can't re-run edits or compare; partial failures corrupt originals.
**Instead:** Keep originals immutable under `pages/`; writes go to `edits/`; result build picks edited if present.

### Anti-Pattern 7: Ignoring pdfcpu API Variability

**What people do:** Hard-code pdfcpu call signatures without checking version.
**Why it's bad:** pdfcpu's API shifts between versions (v0.16.3 pinned); compile/runtime failures.
**Instead:** Verify exact signatures (`api.ExtractImagesFile`, `api.ImportImagesFile`) against installed v0.16.3 at implementation; wrap in `internal/pdfio` so call sites are isolated.

### Anti-Pattern 8: Swallowing Errors / Partial Cleanup

**What people do:** Ignore errors from file deletes during cleanup; leave orphan artifacts.
**Why it's bad:** Disk leak; misleading "deleted" responses.
**Instead:** Log cleanup errors with doc id; optionally run a periodic sweeper (out of scope for MVP, note for future).

### Anti-Pattern 9: Unbounded Concurrency on External Binaries

**What people do:** Spawn Tesseract for every page of every concurrent upload with no limit.
**Why it's bad:** CPU/memory exhaustion; Tesseract is heavy.
**Instead:** A bounded worker pool (e.g., `golang.org/x/sync/semaphore` or a fixed goroutine pool) limiting concurrent OCR processes; configurable via env. Note: this is a recommended addition beyond the minimal plan.

## Scalability Considerations

| Concern | At 1-10 docs (MVP) | At 100s of docs | At 1000s / multi-instance |
|---------|---------------------|-----------------|---------------------------|
| Document state | In-memory map + mutex; fine | Memory grows with images held in RAM (we hold paths, not bytes, so OK-ish) | Not shareable across instances; needs external store |
| OCR throughput | Sequential per request; acceptable | Add bounded worker pool for Tesseract processes | Queue + dedicated workers; horizontal scale |
| Storage | Local FS per process | Disk fills; needs eviction/TTL | Shared object storage (S3) + stateless instances |
| Concurrency safety | Mutex on map sufficient | RWMutex + per-doc locks for parallel edits to same doc | Distributed lock or DB |
| PDF rebuild | Synchronous in request | Cache result.pdf; rebuild only on new edits | Async job + polling |
| Process model | Single binary, single instance | Vertical scale (bigger box) | Stateless API + shared store (requires rework) |

Honest assessment: this architecture is deliberately single-process, in-memory, session-scoped. That is a stated constraint (no auth, no persistence, no multi-tenant). Do not pretend it scales horizontally. If a future milestone needs multi-instance, introduce a `Store` interface and a pluggable backend (disk/DB/object storage) without rewriting handlers — that is the seams to leave now.

## Component Boundaries (file-level)

```text
cmd/server/main.go
  - imports: internal/api, internal/ocr, os, net/http
  - must NOT: import pdfio/edit directly (go through api)

internal/api/handlers.go
  - imports: internal/pdfio, internal/ocr, internal/edit, net/http, sync, ...
  - owns: Store, route registration, orchestration

internal/pdfio/*.go
  - imports: github.com/pdfcpu/pdfcpu, os, path/filepath
  - must NOT: import ocr or edit

internal/ocr/*.go
  - imports: os/exec (cli), gosseract/v2 (gosseract, tagged), image libs if needed
  - must NOT: import pdfio or edit or api

internal/edit/*.go
  - imports: github.com/fogleman/gg, golang.org/x/image, os, path/filepath
  - must NOT: import pdfio or ocr or api
```

## Request Flow Summary (single picture)

```text
Upload PDF
  -> save bytes (size-limited)
  -> pdfio.ExtractImages (largest raster per page)
  -> for each page: ocr.Engine.Recognize -> []Word
  -> Store.Create(doc) under mutex
  -> JSON {docId, pages}

Preview page
  -> Store.Get (mutex) -> path -> http.ServeFile (scoped root)

Edit
  -> validate + resolve words
  -> edit.Erase (sample bg from ring, fill padded rect)
  -> edit.Render (fit font, sample color, baseline draw)
  -> write edits/<id>/<page>.png atomically
  -> Store.MarkEdited (mutex)
  -> preview

Result
  -> for each page: edited if present else original
  -> pdfio.Build (ImportImagesFile) -> result.pdf
  -> stream attachment

Delete
  -> Store.Delete (mutex)
  -> best-effort rm artifact tree (logged)
```

## Sources

- `.planning/codebase/ARCHITECTURE.md` (intended architecture, 2026-10-27)
- `.planning/codebase/STRUCTURE.md` (planned layout, 2026-10-27)
- `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md` (implementation plan)
- `.planning/PROJECT.md` (constraints, decisions)
- Go standard library: `net/http`, `sync`, `os/exec`, `context`, `crypto/rand`
- pdfcpu v0.16.3 (`api.ExtractImagesFile`, `api.ImportImagesFile`)
- fogleman/gg v1.3.0, otiai10/gosseract/v2 v2.4.1 (build-tagged)
- golang.org/x/image v0.47.0, freetype (indirect)