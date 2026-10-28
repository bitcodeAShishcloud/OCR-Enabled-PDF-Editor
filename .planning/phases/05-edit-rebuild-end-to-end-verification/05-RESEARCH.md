# Phase 5: Edit, Rebuild & End-to-End Verification - Research

**Researched:** 2026-10-28
**Domain:** Go raster edit pipeline (`internal/edit`) + API edit/rebuild endpoints + synthetic E2E verification
**Confidence:** HIGH

## Summary

Phase 5 completes the backend. All required libraries are already in `go.mod` (`fogleman/gg` v1.3.0, `golang.org/x/image` v0.47.0, `pdfcpu` v0.16.3) and no new module dependencies are needed. A working prototype was run in this session: `gg` renders a name to PNG, Tesseract OCRs it with accurate bboxes, the erase (ring-sampled background fill) + binary-search font fit + darkest-pixel text color sampling + baseline placement pipeline produces an edited PNG that Tesseract re-reads as the new text, and `pdfio.BuildPDF` round-trips that PNG through a PDF with the new text surviving re-OCR.

The main correctness risks are (1) baseline/descent handling so glyphs fit vertically, (2) `font.Face` is explicitly not thread-safe (must be created per operation, never shared), (3) the currently-immutable `Document`/`Page` model must gain a mutable current-image path plus a per-document mutex for edits/rebuilds, and (4) rebuild must snapshot image paths under the lock and build outside it. The E2E test should render at a generous size (e.g. 48 pt on a 400x120 image) so Tesseract is reliable.

**Primary recommendation:** Implement `internal/edit` as pure functions over `*image.RGBA` (Erase, SampleBackground, SampleTextColor, FitFontSize, RenderText) with a small `Rect`/bbox convention matching Tesseract's `X0,Y0,X1,Y1`, add `EditedPath`-equivalent current path by overwriting `Page.ImagePath` plus a `sync.Mutex` on `Document`, add `Edit` and `Result` handlers, register two routes, and add a synthetic E2E test that skips when Tesseract is absent.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Sample background color from the pixel ring around the bbox; fill the padded bbox. Reversible.
- **D-02:** Auto-fit font size to bbox width via binary search. Reversible.
- **D-03:** Sample text color from old glyphs' darkest pixels. Reversible.
- **D-04:** Baseline-aligned to bbox bottom using font metrics. Reversible.
- **D-05:** Load TTF from `FONT_PATH` (default Arial). Reversible.
- **D-06:** `POST /api/documents/{id}/edits` accepts page, oldText or wordIds, newText, optional font/size/color. Returns edited preview. Reversible.
- **D-07:** `GET /api/documents/{id}/result.pdf` rebuilds from current edited images and streams. Reversible.
- **D-08:** Synthetic: gg draws a name onto an image  pdfcpu imports to PDF  upload  OCR  edit  rebuild  re-OCR verifies new text. Reversible.

### the agent's Discretion
- Preview response shape (image bytes vs URL).
- Color sampling heuristic details.

### Deferred Ideas (OUT OF SCOPE)
None - discussion stayed within phase scope.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EDIT-01 | Erase original text by sampling background color from the bbox ring and filling the padded bbox | `internal/edit.Erase` + `SampleBackground` (median per-channel ring sampling, dark-pixel exclusion, padding clamp) |
| EDIT-02 | Render replacement text with font size auto-fitted to bbox width | `internal/edit.FitFontSize` (binary search on width and height) |
| EDIT-03 | Render uses text color sampled from the original glyphs' darkest pixels | `internal/edit.SampleTextColor` (darkest-pixel median, fallback black) |
| EDIT-04 | New text is baseline-aligned to the bbox bottom | `Metrics.Ascent/Descent` baseline = `bbox.Max.Y - Descent`, clamped to ascent |
| EDIT-05 | Load TTF font from `FONT_PATH` (default `C:\Windows\Fonts\arial.ttf`) | `gg.LoadFontFace(path, points)`; Handler carries `FontPath` |
| EDIT-06 | Edit endpoint accepts page, oldText or wordIds, newText, and optional font/size/color overrides | `POST /api/documents/{id}/edits` JSON body; word index resolution; override application |
| API-03 | `POST /api/documents/{id}/edits` applies an edit and returns the edited preview | New `Edit` handler + route registration in `main.go` |
| API-04 | `GET /api/documents/{id}/result.pdf` rebuilds and streams the final PDF | New `Result` handler using `pdfio.BuildPDF` + `http.ServeFile` |
| TEST-02 | Unit tests for font-fit function | `FitFontSize` is pure-ish (takes font path) and directly testable |
| TEST-03 | Synthetic end-to-end test | Render name with `gg`, import via `pdfcpu`, upload, OCR, edit, rebuild, re-OCR |

</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Erase / render / color sampling | API / Backend (CPU, image) | - | Pure in-process raster work on `*image.RGBA`; no external service |
| Font loading & metrics | API / Backend | OS filesystem (TTF) | `gg.LoadFontFace` reads `FONT_PATH`; `font.Face` created per call |
| Edit orchestration (word lookup, image mutation, persistence) | API / Backend | - | Handler owns document state and per-doc mutex |
| Rebuild (images -> PDF) | API / Backend | `internal/pdfio` | `pdfio.BuildPDF` (pdfcpu) runs in-process |
| PDF streaming | API / Backend | OS filesystem | `http.ServeFile` on rebuilt `result.pdf` |
| Concurrency control | API / Backend | - | Per-document `sync.Mutex` + store mutex |
| E2E verification | Test (Go `testing`) | Tesseract binary | Skips when Tesseract/font unavailable |

## Standard Stack

No new dependencies. All already in `go.mod`:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/fogleman/gg` | v1.3.0 | `Context` raster ops: `NewContextForRGBA`, `DrawRectangle`, `SetColor`, `DrawString`, `MeasureString`, `LoadFontFace`, `Image`, `SavePNG` | Already chosen; wraps freetype + x/image |
| `golang.org/x/image` | v0.47.0 | `draw`, `font` (Face, Metrics, MeasureString), `color` helpers | Transitive dep of gg; metrics for baseline |
| `github.com/golang/freetype` | v0.0.0-20170609003504-e2365dfdc4a0 | `truetype.Parse`, `truetype.NewFace` (via gg) | Font rasterization; do not hand-roll |
| `github.com/pdfcpu/pdfcpu` | v0.16.3 | Rebuild via `internal/pdfio.BuildPDF` | Already wraps `ImportImagesFile` |
| Go stdlib `image`, `image/color`, `image/draw`, `image/png` | Go 1.26.5 | Pixel access, conversion to RGBA, PNG encode/decode | Standard |
| Go stdlib `sync` | Go 1.26.5 | Per-document mutex | Race-detector friendly |
| Go stdlib `encoding/json` | Go 1.26.5 | Edit request/response | Standard |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `ocrpdf/internal/ocr` | local | `Word` bbox + `Engine.Recognize` for E2E re-OCR | Word lookup by `oldText`/`wordIds` |
| `ocrpdf/internal/pdfio` | local | `ExtractImages`, `BuildPDF` | Upload (existing) and result (new) |
| `ocrpdf/internal/api` | local | `Store`, `Document`, `Page`, handlers | Extended in this phase |

**Installation:** none - all dependencies present. Run `go mod tidy` only if a new import is added (none expected).

## Package Legitimacy Audit

No external packages are introduced in this phase. All libraries referenced above are already declared in `go.mod`/`go.sum` and were verified in earlier phases. No legitimacy check required.

## Architecture Patterns

### Edit Pipeline Data Flow

```
POST /api/documents/{id}/edits {page, oldText|wordIds, newText, font?, size?, color?}
        |
        v
  Store.Get(id)  --404 if missing
        |
        v
  doc.mu.Lock()  (held for whole edit)
        |
        v
  resolve target words on doc.Pages[page-1].Words  --> union bbox (image px)
        |
        v
  load current page image (Page.ImagePath) -> *image.RGBA
        |
        v
  sample text color from bbox pixels (darkest)   -- before erase
  sample background color from ring around bbox  -- before erase
        |
        v
  Erase: fill padded bbox rect with bg color
        |
        v
  FitFontSize(fontPath, newText, bboxW, bboxH)  (if size not provided)
        |
        v
  RenderText: LoadFontFace, SetColor(textColor), DrawString at baseline
        |
        v
  png.Encode -> doc dir "edited_p{page}.png"; Page.ImagePath = new path
        |
        v
  doc.mu.Unlock()
        |
        v
  200 image/png (edited page bytes)

GET /api/documents/{id}/result.pdf
        |
        v
  Store.Get(id) --404
  doc.mu.Lock(); snapshot []imagePaths from Page.ImagePath; doc.mu.Unlock()
        |
        v
  pdfio.BuildPDF(ctx, paths, docDir/result.pdf)   (outside lock)
        |
        v
  200 application/pdf via http.ServeFile
```

### Recommended Project Structure (additions)

```
internal/edit/
├── edit.go        # Rect helpers, shared types, color parsing
├── erase.go       # Erase, SampleBackground
├── render.go      # RenderText, FitFontSize
└── color.go       # SampleTextColor, luminance, median helpers
internal/api/
├── handlers.go    # + Edit, Result methods
├── store.go       # + sync.Mutex on Document
```

### Pattern 1: Erase with ring-sampled background

**What:** Build an annulus around the word bbox (expand by `ring`, e.g. 6 px), collect pixels outside the bbox, exclude very dark ones (likely text/ink), and take the per-channel median. Fill the bbox expanded by `pad` (e.g. 4 px) with that color, clamped to image bounds.

**Why:** Median is robust to a few dark edge pixels; padding covers anti-aliased glyph fringes.

**Verified prototype result:** white background sampled as `{255,255,255}` on the synthetic image; new text re-OCRs correctly.

### Pattern 2: Font fit via binary search

**What:** Search font size in `[4, min(boxH, 200)]` points. For each candidate load the face, measure advance width with `font.MeasureString`, and read `face.Metrics().Height`. Accept if `width <= boxW && heightPx <= boxH`. Return the largest accepted size.

**When to use:** Always when `fontSize` is omitted; honor an explicit override but still clamp to box height if it overflows (or reject with 400).

**Verified:** 48 px bbox height -> fitted 35 pt for "Jane" in a 100 px-wide bbox in the prototype.

### Pattern 3: Baseline from font metrics

**What:** `font.Face.Metrics()` returns `Ascent`, `Descent` (positive magnitude), `Height` in `fixed.Int26_6` (1/64 px). Baseline `y = bbox.Max.Y - Descent/64`. Clamp: if `baseline - Ascent/64 < bbox.Min.Y`, set `baseline = bbox.Min.Y + Ascent/64`. Draw with `gg.Context.DrawString(text, bbox.Min.X, baseline)` (gg's DrawString places the baseline at `(x,y)`).

**Why:** Drawing at `bbox.Max.Y` directly (Pitfall 6) makes descenders overflow and ascenders clip; using metrics places the glyph cell inside `[bbox.Min.Y, bbox.Max.Y]`.

### Pattern 4: Thread-safe font usage

**What:** `gg.LoadFontFace` (and `truetype.NewFace`) return a `font.Face` whose doc comment states it is **not safe for concurrent use**. Create a new face per operation (`FitFontSize` and `RenderText` each call `gg.LoadFontFace`); never store a shared face on the Handler or in a package global. `gg.Context` is also per-image (`NewContextForRGBA`), so create one per edit.

### Pattern 5: Mutable document state with per-doc mutex

**What:** Add `sync.Mutex` to `Document` (zero value is usable, so existing tests that build `&Document{...}` still compile). Store the "current" page image by overwriting `Page.ImagePath` after a successful edit (original extraction path is replaced; the edited PNG lives in the same temp dir). Access pattern:
- `Store.Get` returns `*Document` under the store mutex (released immediately).
- Callers take `doc.mu.Lock()` for the duration of an edit or for a path snapshot during rebuild/preview.
- `GetPage` copies the path under `doc.mu`, then serves the file outside the lock.
- `Result` snapshots paths under `doc.mu`, unlocks, then builds + serves outside the lock.

Lock ordering: store mutex -> doc mutex, never reverse. Never hold doc mutex during Tesseract/pdfcpu calls except where correctness requires (edit holds it for the whole mutation, which is acceptable for v1).

### Pattern 6: Rebuild invalidates implicitly

**What:** There is no cached result PDF. `Result` always rebuilds from current `Page.ImagePath` values into `doc.Dir/result.pdf` (truncated first by `pdfio.BuildPDF`). An edit changes `Page.ImagePath`, so the next rebuild reflects it automatically. `os.Remove` of a stale `result.pdf` is handled inside `BuildPDF`.

### Anti-Patterns to Avoid

- **Shared `font.Face` across goroutines** - breaks race detector and corrupts glyph caches; always create per call.
- **Drawing baseline at `bbox.Max.Y` or `bbox.Min.Y`** - clips ascenders/descenders; use metrics.
- **Fitting font size on width only** - tall glyphs overflow the bbox height; enforce both dimensions.
- **Holding the store mutex during OCR/pdfcpu** - serializes all requests and risks deadlock; hold only for map access or the whole per-doc edit.
- **Appending to an existing `result.pdf`** - `pdfio.BuildPDF` already removes first; do not bypass it.
- **Building paths from the raw doc ID without validation** - ID is a UUID from `newUUID`; still `filepath.Clean` and contain-check against `doc.Dir`.
- **Mutating `Document.Pages` without `doc.mu`** - race with concurrent preview/rebuild.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Font rasterization / metrics | Custom TTF parser | `gg.LoadFontFace` + `golang.org/x/image/font` | Hinting, kerning, glyph masks already handled |
| PDF rebuild | Custom PDF writer | `pdfio.BuildPDF` (pdfcpu) | Page dims, truncation, encoding handled |
| TSV/OCR | Custom OCR | Existing `ocr.Engine` | Tesseract invocation + parsing done |
| UUID generation | Custom random ID | `crypto/rand` (existing `newUUID`) | Already implemented, v4 format |
| Multipart parsing | Manual body parsing | `r.ParseMultipartForm` + `r.FormFile` | Spills to disk, standard |
| Color hex parsing | Heavy color library | Small helper using `strconv`/`fmt.Sscanf` | Only need #RGB/#RRGGBB |
| Median computation | Sorting library | Inline sort (slices are tiny) | Avoids new dependency |
| File serving | Manual `io.Copy` | `http.ServeFile` | Range requests, content type, modtime |

**Key insight:** The only genuinely new logic is the color-sampling heuristics and the binary-search fit; everything else (font, PDF, OCR, HTTP) is already solved by existing deps. Keep `internal/edit` dependency-free except for `gg` and stdlib.

## Common Pitfalls

### Pitfall 1: Baseline placement clips text
**What goes wrong:** `DrawString` baseline at `bbox.Max.Y` overflows descenders; at `bbox.Min.Y` clips ascenders.
**How to avoid:** `baseline = bbox.Max.Y - Descent/64`, clamp to `bbox.Min.Y + Ascent/64`. (Verified in prototype.)

### Pitfall 2: Font size overflow in height
**What goes wrong:** Fitting only on width yields glyphs taller than the bbox.
**How to avoid:** `FitFontSize` rejects candidates where `Metrics.Height/64 > boxH`.

### Pitfall 3: `font.Face` shared across goroutines
**What goes wrong:** Race detector failure and corrupted glyph caches.
**How to avoid:** Create a face per call; never store on Handler.

### Pitfall 4: Sampling text color after erase
**What goes wrong:** Background fill overwrites glyphs; color sampling returns background.
**How to avoid:** Sample text color and background color from the original image before erasing.

### Pitfall 5: Non-uniform backgrounds
**What goes wrong:** Ring contains text/gradient pixels; mean produces wrong fill.
**How to avoid:** Median per channel, exclude pixels darker than a threshold, clamp fill rect to bounds, pad the bbox.

### Pitfall 6: Word resolution ambiguity
**What goes wrong:** `oldText` matches multiple words or none; `wordIds` out of range.
**How to avoid:** Return 400 with a clear message; prefer `wordIds` when both are supplied; compute union bbox for multiple ids.

### Pitfall 7: Concurrent edits on same document
**What goes wrong:** Two edits mutate the same image; rebuild races.
**How to avoid:** Hold `doc.mu` for the full edit; snapshot paths under the lock for rebuild.

### Pitfall 8: Windows file locks on rebuild
**What goes wrong:** `os.Remove` fails if a handle is open.
**How to avoid:** Close all readers/writers before removal; `pdfio.BuildPDF` already removes first and `http.ServeFile` opens its own handle.

### Pitfall 9: E2E font/OCR mismatch
**What goes wrong:** Tesseract misreads stylized or tiny text.
**How to avoid:** Use Arial at ~48 pt on a ~400x120 white image; assert OCR contains the expected word with a tolerance; skip if Tesseract/font missing.

### Pitfall 10: Store immutability assumption
**What goes wrong:** Existing comment says "Document is immutable after insertion"; Phase 5 needs mutation.
**How to avoid:** Remove/relax that comment; add `sync.Mutex` and document the locking contract; update `TestStoreConcurrent` expectations if needed (it only uses map ops).

## Code Examples

### gg erase + render (verified pattern)

```go
// Source: prototype run in this session against gg v1.3.0
dc := gg.NewContextForRGBA(img) // mutates img in place
dc.SetColor(bg)                 // color.Color
dc.DrawRectangle(float64(x0), float64(y0), float64(w), float64(h))
dc.Fill()
```

`gg.Context.DrawRectangle` adds a sub-path; call `Fill()` (defined in context.go path ops) to paint. `SetColor` sets both fill and stroke.

### Load font face and measure

```go
// Source: gg@v1.3.0/util.go:132-148
face, err := gg.LoadFontFace(fontPath, points)
// face.Metrics() -> Ascent, Descent, Height (fixed.Int26_6, /64 for px)
adv := font.MeasureString(face, text) // fixed.Int26_6
widthPx := float64(adv >> 6)
```

### Binary-search font fit

```go
func FitFontSize(fontPath, text string, boxW, boxH float64) (float64, error) {
    if text == "" { return 4, nil }
    lo, hi := 4.0, boxH
    if hi > 200 { hi = 200 }
    if hi < 4 { hi = 4 }
    best := 4.0
    for i := 0; i < 40; i++ {
        mid := (lo + hi) / 2
        face, err := gg.LoadFontFace(fontPath, mid)
        if err != nil { return 0, err }
        w := float64(font.MeasureString(face, text) >> 6)
        h := float64(face.Metrics().Height) / 64.0
        if w <= boxW && h <= boxH {
            best = mid; lo = mid
        } else {
            hi = mid
        }
    }
    return best, nil
}
```

### Baseline placement

```go
face, _ := gg.LoadFontFace(fontPath, size)
m := face.Metrics()
baseline := float64(bbox.Max.Y) - float64(m.Descent)/64.0
ascent := float64(m.Ascent) / 64.0
if baseline-ascent < float64(bbox.Min.Y) {
    baseline = float64(bbox.Min.Y) + ascent
}
dc := gg.NewContextForRGBA(img)
dc.SetColor(textColor)
_ = dc.LoadFontFace(fontPath, size)
dc.DrawString(newText, float64(bbox.Min.X), baseline)
```

### Color sampling helpers (median)

```go
func luminance(c color.RGBA) float64 {
    return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

// ring: outer = bbox.Inset(-ring); collect pixels in outer not in bbox,
// skip pixels with luminance < 80 (ink), median per channel.
// text color: pixels inside bbox with luminance < bgLum - 30; if fewer than 5,
// take the darkest 10%; median per channel; fallback color.RGBA{0,0,0,255}.
```

### Edit handler skeleton

```go
type editRequest struct {
    Page    int      `json:"page"`
    OldText string   `json:"oldText"`
    WordIDs []int    `json:"wordIds"`
    NewText string   `json:"newText"`
    FontPath string  `json:"fontPath"`
    FontSize *float64 `json:"fontSize"`
    Color   string   `json:"color"`
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) { h.edit(w, r) }

func (h *Handler) edit(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    var req editRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, 400, "invalid json"); return }
    if req.Page < 1 { writeError(w, 400, "page required"); return }
    doc, ok := h.Store.Get(id)
    if !ok { writeError(w, 404, "document not found"); return }
    doc.mu.Lock()
    defer doc.mu.Unlock()
    if req.Page > len(doc.Pages) { writeError(w, 400, "page not found"); return }
    // resolve words -> union bbox
    // sample colors + erase + fit + render on a copy
    // save PNG, update Page.ImagePath
    // write edited PNG bytes to response
}
```

### Result handler skeleton

```go
func (h *Handler) Result(w http.ResponseWriter, r *http.Request) { h.result(w, r) }

func (h *Handler) result(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    doc, ok := h.Store.Get(id)
    if !ok { writeError(w, 404, "document not found"); return }
    doc.mu.Lock()
    paths := make([]string, len(doc.Pages))
    for i, p := range doc.Pages { paths[i] = p.ImagePath }
    doc.mu.Unlock()
    out := filepath.Join(doc.Dir, "result.pdf")
    if err := pdfio.BuildPDF(r.Context(), paths, out); err != nil {
        writeError(w, 500, "rebuild failed"); return
    }
    w.Header().Set("Content-Type", "application/pdf")
    http.ServeFile(w, r, out)
}
```

### Store change

```go
// internal/api/store.go
type Page struct {
    Number    int
    ImagePath string
    Words     []ocr.Word
}
type Document struct {
    mu    sync.Mutex
    ID    string
    Dir   string
    Pages []Page
}
```

Relax the "immutable after insertion" comment; document the locking contract.

### main.go route registration

```go
mux.HandleFunc("POST /api/documents/{id}/edits", h.Edit)
mux.HandleFunc("GET /api/documents/{id}/result.pdf", h.Result)
// Handler gets FontPath: cfg.FontPath
```

### Synthetic E2E test (TEST-03)

```go
// internal/api/e2e_edit_test.go (or handlers_test.go)
func TestEditRebuildRoundTrip(t *testing.T) {
    if _, err := os.Stat(`C:\Program Files\Tesseract-OCR\tesseract.exe`); err != nil { t.Skip(...) }
    if _, err := os.Stat(`C:\Windows\Fonts\arial.ttf`); err != nil { t.Skip(...) }
    dir := t.TempDir()
    // 1. render name
    imgPath := filepath.Join(dir, "src.png")
    dc := gg.NewContext(400, 120)
    dc.SetColor(color.White); dc.Clear()
    dc.SetColor(color.Black)
    dc.LoadFontFace(fontPath, 48)
    dc.DrawString("John", 20, 70)
    dc.SavePNG(imgPath)
    // 2. import to PDF
    pdfPath := filepath.Join(dir, "in.pdf")
    imp := pdfapi.DefaultImportConfig(); imp.PageDim = &types.Dim{Width: 400, Height: 120}
    pdfapi.ImportImagesFile(ctx, []string{imgPath}, pdfPath, imp, model.NewStatelessConfiguration())
    // 3. upload + OCR
    // 4. POST edit {page:1, oldText:"John", newText:"Jane"}
    // 5. GET result.pdf -> extract -> OCR -> assert contains "Jane"
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Baseline at bbox bottom | Metrics-based baseline | This phase | Prevents clipping |
| Width-only font fit | Width + height fit | This phase | Prevents vertical overflow |
| Immutable Document | Per-doc mutex + mutable current image path | This phase | Enables edits/rebuild |

**Deprecated/outdated:**
- "Document is immutable after insertion" comment in `store.go` - must be relaxed for this phase.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Preview response returns raw PNG bytes (not JSON URL) | Architecture Patterns | Low - planner can choose JSON; tests only need consistency |
| A2 | `wordIds` are zero-based indices into `Page.Words` | Code Examples | Medium - if planner expects 1-based, adjust |
| A3 | Editing overwrites `Page.ImagePath` with the edited PNG path | Pattern 5 | Medium - if original path must be preserved, add `EditedPath` field instead |
| A4 | Ring size 6 px and padding 4 px are acceptable defaults | Pattern 1 | Low - heuristic; configurable later |
| A5 | Darkest-pixel median yields the original glyph color reliably | Pattern text color | Medium - gradient/colored text may need override (supported via `color`) |
| A6 | E2E uses 48 pt Arial on 400x120 white background for OCR reliability | TEST-03 | Medium - OCR accuracy varies; skip if Tesseract absent |
| A7 | `FontPath` is added to `api.Handler` (not read from env inside edit) | Standard Stack | Low - consistent with main.go wiring |

## Open Questions

1. **Multiple occurrences of `oldText` on one page.**
   - What we know: prototype replaces one bbox.
   - What's unclear: whether to replace first match or all.
   - Recommendation: replace first match by default; return 400 if ambiguous; prefer `wordIds` for precision.

2. **Phrase (multi-word) replacement.**
   - What we know: union bbox works for rendering a single new string.
   - What's unclear: whether to require contiguous wordIds.
   - Recommendation: support union bbox of any selected wordIds; do not attempt automatic phrase matching in v1.

3. **Explicit `fontSize` override validation.**
   - What we know: override should be honored.
   - What's unclear: whether to reject if it overflows height.
   - Recommendation: honor override but clamp to box height (shrink if overflow) to avoid clipping.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build/test | ✓ | 1.26.5 | - |
| Tesseract OCR | OCR + E2E | ✓ | 5.5.0 (Windows default path) | Skip E2E test |
| Arial TTF | Font rendering | ✓ | `C:\Windows\Fonts\arial.ttf` | `FONT_PATH` override |
| pdfcpu | Rebuild | ✓ | v0.16.3 (Go module) | - |

**Missing dependencies with no fallback:** none for the default build.
**Missing dependencies with fallback:** Tesseract/font absence skips the synthetic E2E test.

## Security Domain

Security enforcement is enabled (ASVS L1). Phase 5 adds input handling on the edit/rebuild endpoints.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | Out of scope (no auth) |
| V3 Session Management | no | Stateless per request |
| V4 Access Control | no | No user accounts |
| V5 Input Validation | yes | Validate JSON fields (page range, wordIds indices, newText length, optional color hex, optional font path), size caps, reject unknown doc ids with 404 |
| V6 Cryptography | no | No crypto in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal via doc ID | Tampering | UUID v4 from `newUUID`; `filepath.Clean` + contain check against `doc.Dir` |
| Oversized request body | DoS | `http.MaxBytesReader` (reuse existing cap) |
| Font path injection (`fontPath` override) | Tampering | Validate file exists and is a readable TTF; restrict to `FONT_PATH` directory or reject arbitrary paths (recommended: ignore client `fontPath` unless it points under a configured root; v1 can accept but `os.Stat` + parse) |
| Color string injection | Tampering | Strict hex parser (`#RGB`/`#RRGGBB`) |
| Race on document state | Tampering | Per-doc mutex |
| Leftover temp files | DoS | Reuse per-doc temp dir; `os.RemoveAll` on delete |

## Sources

### Primary (HIGH confidence)
- `internal/ocr/ocr.go:5-14` - `Word` struct and `Engine` interface (verified this session)
- `internal/api/store.go` - `Page`, `Document`, `Store` (verified this session)
- `internal/api/handlers.go` - handler patterns, DTOs (verified this session)
- `internal/pdfio/build.go` - `BuildPDF` truncation + PageDim (verified this session)
- `github.com/fogleman/gg@v1.3.0/util.go:132-148` - `LoadFontFace` signature + thread-safety note (verified this session)
- `github.com/fogleman/gg@v1.3.0/context.go:679-695` - `SetFontFace`, `LoadFontFace`, `FontHeight` (verified this session)
- `golang.org/x/image@v0.47.0/font/font.go` - `Face`, `Metrics`, `MeasureString` (verified this session)
- Prototype run in this session: render -> OCR -> erase -> fit -> render -> re-OCR -> PDF round trip (all succeeded)

### Secondary (MEDIUM confidence)
- `.planning/research/PITFALLS.md` - baseline/color/race pitfalls (project doc)
- `.planning/phases/04-.../04-RESEARCH.md` - handler/store patterns (project doc)

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all deps already in go.mod and exercised in prototype
- Architecture: HIGH - patterns verified by working prototype and existing code
- Pitfalls: HIGH - derived from verified library docs + project PITFALLS.md

**Research date:** 2026-10-28
**Valid until:** 2026-11-27 (30 days; stable Go/pdfcpu/Tesseract stack)
