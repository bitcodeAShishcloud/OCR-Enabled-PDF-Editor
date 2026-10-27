# Domain Pitfalls

**Domain:** Go HTTP backend for scanned-PDF OCR text replacement
**Researched:** 2026-10-27
**Confidence:** MEDIUM-HIGH (verified against pdfcpu v0.16.3 pkg.go.dev, Tesseract TSV spec, gosseract windows notes; Go concurrency/memory guidance from standard library docs)

## Critical Pitfalls

Mistakes that cause rewrites or major issues.

### Pitfall 1: pdfcpu `ExtractImagesFile` returns only an error, not filenames

**What goes wrong:** `api.ExtractImagesFile(ctx, inFile, outDir, selectedPages, conf)` writes every embedded image resource to `outDir` with pdfcpu-generated names and returns only `error`. A scanned page can contain multiple XObjects (soft masks, thumbnails, OCG layers, duplicate images), so "one image per page" is false. Implementing "largest image per page" by assuming a single output file silently picks the wrong image or fails when zero/multiple files appear.

**Why it happens:** pdfcpu's extraction is resource-oriented, not page-oriented. The plan's "largest image per page" must be derived by the caller.

**Consequences:** OCR runs on a mask/thumbnail; rebuild uses a tiny image; flaky failures on PDFs with masks.

**Prevention:**
- After extraction, enumerate `outDir`, stat each file, decode image dimensions, group by page, and select the largest by pixel area (`width*height`).
- Clean `outDir` per document before extraction to avoid stale files from a previous run.
- Treat `*pdfcpu.UnsupportedResourceError` (skip mode) as "no usable image" and fail the document with a clear error rather than proceeding with partial data.
- Wrap pdfcpu calls behind `internal/pdfio` with integration tests on fixture PDFs.

**Detection:** Unit/integration test counting extracted files per page; log selected image name + dimensions.

### Pitfall 2: `ImportImagesFile` appends to an existing output file

**What goes wrong:** `api.ImportImagesFile(ctx, imgFiles, outFile, imp, conf)` creates `outFile` if necessary but appends pages when it already exists. Re-running a rebuild (or a partial failure retry) produces a PDF with duplicated/trailing pages.

**Why it happens:** The function is append-oriented by design (it can append to an existing PDF).

**Consequences:** Result PDF has N+pages on retry; E2E OCR of result returns stale/extra text.

**Prevention:**
- Always `os.Remove(outFile)` (ignore `IsNotExist`) before calling `ImportImagesFile`, or write to a fresh temp path.
- Never reuse a result path across rebuilds without truncating.
- Validate with `api.ValidateImportImagesOutput(imgFiles, outFile)` to catch aliasing an input image as the output file.

**Detection:** Page count assertion in tests; check output page count equals input image count.

### Pitfall 3: Default `Import` config defaults to A4 and full-page stretch

**What goes wrong:** `pdfcpu.DefaultImportConfig()` sets `PageDim = PaperSize["A4"]`, `Pos = Full`, `Scale = 0.5`. In `Full` position the image is scaled to the whole page dimensions, ignoring aspect ratio and the scan's real size. A 2550x3300 px scan becomes a stretched A4 page; text aspect distorts and OCR of the rebuilt PDF may fail.

**Why it happens:** Defaults are tuned for "import arbitrary images onto paper", not "lossless scan rebuild".

**Consequences:** Visually distorted output; replacement text metrics no longer match.

**Prevention:**
- Build the `*pdfcpu.Import` explicitly: set `PageDim` to the original page dimensions (from `api.PageDimsFile` on the source PDF, or image pixel size converted at the scan DPI) and `Pos = types.Full` only when aspect ratios match.
- Alternatively set `DPI` and a non-`Full` position so pdfcpu converts pixel dimensions to points; verify `PageDim != nil` (function errors otherwise).
- Preserve the original PDF's page size per page rather than a single global size.

**Detection:** Compare output page dimensions to input page dimensions in tests.

### Pitfall 4: Tesseract TSV parsing breaks on header, empty lines, and `-1` confidence

**What goes wrong:** Tesseract's TSV has 12 columns (`level, page_num, block_num, par_num, line_num, word_num, left, top, width, height, conf, text`). Rows at levels 1-4 carry `conf = -1` and empty `text`. Naive `strings.Split` on tabs fails on trailing empty fields, the header row, blank lines, and rows where `text` is empty (field count drops). Confidence `-1` must be filtered or garbage words leak in.

**Why it happens:** TSV is not strictly rectangular in hand-rolled parsers; empty trailing fields are elided by naive split.

**Consequences:** Parse errors on every page with no words; phantom words with confidence -1; off-by-one column reads.

**Prevention:**
- Use `encoding/csv` with `Comma = '\t'` and `FieldsPerRecord = -1` (variable).
- Skip the header row (first field == "level") and blank lines.
- Only accept rows where `level == "5"` and `conf >= 0`.
- Require at least 11 fields; treat missing `text` as empty string.
- Parse numbers with `strconv.ParseFloat` (Tesseract emits floats like `96.063751`).
- Trim only the `text` field's trailing newline; keep internal spaces.
- Unit-test with golden TSV including header, empty text, `-1` rows, and a blank line.

**Detection:** Fuzz/property tests on TSV samples; assert word count on a known fixture.

### Pitfall 5: Image coordinate-system mismatch (Tesseract vs pdfcpu vs gg)

**What goes wrong:** Tesseract's `left/top/width/height` are in image pixels with origin at top-left. pdfcpu extracts the embedded raster as-is; gg also uses a top-left origin. Mixing PDF point coordinates (bottom-left origin) into the image pipeline flips text vertically or places replacements off-page. Rotated PDF pages make the mismatch worse.

**Why it happens:** Three different coordinate conventions exist; the pipeline must stay in one space.

**Consequences:** Replacement rendered upside-down or outside the page; color sampling reads the wrong pixels.

**Prevention:**
- Keep erase/render entirely in image pixel space; never convert to PDF points inside `internal/edit`.
- Map Tesseract bbox directly: `x0=left, y0=top, x1=left+width, y1=top+height`.
- If page dimensions from pdfcpu are needed, treat them as image pixels, not points.
- Handle PDF `/Rotate` by checking the extracted image's orientation; if rotation is present, either reject or rotate consistently and document it.
- Add a coordinate round-trip test: OCR bbox -> erase -> render -> re-OCR.

**Detection:** Visual inspection of preview PNG; automated re-OCR of edited result.

### Pitfall 6: Font baseline alignment done by drawing at bbox bottom

**What goes wrong:** gg's `DrawString` places the text baseline at the given `(x, y)`. Drawing at `y = bbox.bottom` makes descenders (g, j, y) extend below the bbox and ascenders get clipped at the top; drawing at `y = bbox.top` puts the baseline at the top. Binary-searching font size by width alone can produce glyphs taller than the bbox.

**Why it happens:** Confusing the bbox's bottom edge with the baseline; ignoring ascent/descent.

**Consequences:** Text clipped or overflowing; inconsistent vertical placement; re-OCR misses glyphs.

**Prevention:**
- Load the TTF via `gg.LoadFontFace` and read metrics from `golang.org/x/image/font` (Ascent, Descent, Height).
- Compute baseline as `y0 + ascent` (or `y1 - descent` where descent is positive magnitude) and clamp so the full glyph cell fits `[y0, y1]`.
- Binary-search font size on width AND enforce `metrics.Height <= bbox height`; shrink if height overflows.
- Sample text color from the darkest pixels of the original glyphs inside the bbox (not the whole page); fall back to black if sampling is unreliable (e.g., max channel spread too small).
- Allow explicit `color`/`size` overrides from the API to escape heuristic failures.

**Detection:** Render a known string and measure its bounding box; unit test the fit function.

### Pitfall 7: Background erase sampling fails on non-uniform scans

**What goes wrong:** Sampling a single color from the pixel ring around the bbox assumes a uniform background. Gradients, paper texture, shadows, bleed-through, or overlapping glyphs in the ring produce a wrong fill color, leaving a visible rectangle.

**Why it happens:** The erase heuristic is a local mean on a thin ring.

**Consequences:** Visible boxes around replacements; "dirty" output; failed visual consistency requirement.

**Prevention:**
- Sample a wider ring (e.g., 4-8 px) and take the median (or mode) per channel, not the mean.
- Exclude pixels whose distance to the bbox edge is below a threshold and pixels that are too dark (likely text) from the sample.
- Sample multiple concentric rings and blend; prefer the outermost clean ring.
- Make erase padding and sampling configurable; support a "no-erase" option and an explicit background color override.
- Clamp the fill rectangle to image bounds; pad the bbox (e.g., 2-4 px) to cover anti-aliased edges.
- Document the limitation; add a test with a gradient background.

**Detection:** Synthetic tests with gradient/shadow backgrounds; manual visual review.

### Pitfall 8: CGO build-tag leakage and unbuildable gosseract variant

**What goes wrong:** If `gosseract.go` is not guarded by `//go:build gosseract`, the default build tries to compile CGO and fails on machines without headers. Conversely, forgetting the tag on tests or on a helper file breaks `go test ./...`. On Windows, gosseract needs MinGW GCC plus Tesseract/Leptonica headers built with MinGW; MSVC-built Tesseract (the common Windows installer) is ABI-incompatible, so linking fails even with headers.

**Why it happens:** Build constraints are easy to miss on secondary files; Windows toolchain mismatch is non-obvious.

**Consequences:** Default build breaks; CI matrix red; gosseract backend never exercised.

**Prevention:**
- Put `//go:build gosseract` at the very top of every gosseract-only file (including `_test.go`).
- Keep the CLI backend as the default; select via `OCR_BACKEND` and validate at startup that the requested backend is actually compiled in.
- Build the gosseract variant with `CGO_ENABLED=1` and document exact header install steps per OS (Linux: `libtesseract-dev libleptonica-dev`; Windows: MinGW + headers built with MinGW, not the MSVC installer).
- Add a CI job running `go build -tags gosseract ./...` and `go test -tags gosseract ./...`.
- Default CI builds with `CGO_ENABLED=0` to guarantee no accidental CGO dependency.

**Detection:** `go build ./...` on a clean machine; `go vet` with tags.

### Pitfall 9: In-memory document store leaks and races

**What goes wrong:** A mutex-protected `map[string]*Document` grows without bound; `DELETE` removes the record but leaves files on disk; copying a struct containing a `sync.Mutex` (or passing by value) breaks synchronization; holding the lock during OCR/file I/O serializes all requests and risks deadlocks; slices stored in the document are mutated concurrently.

**Why it happens:** In-memory stores look simple but need disciplined lifecycle and locking.

**Consequences:** OOM on long-running processes; data races under `go test -race`; stale files; lost edits.

**Prevention:**
- Use a single `sync.RWMutex` guarding the map; take a read lock to look up, then a per-document mutex for edits (lock ordering: store lock -> document lock, never reverse).
- Never copy mutexes: store `*Document` pointers and keep the mutex as a field of the pointed-to struct.
- Use `defer Unlock()` immediately after `Lock()` so early returns/panics release.
- On `DELETE`, remove the map entry under the lock, then delete files outside the lock; use `os.RemoveAll` on the document's temp dir.
- Add TTL/eviction or a max-document cap for safety.
- Use unguessable UUIDs for doc IDs; never accept user-supplied path segments.
- Run `go test -race ./...` in CI.

**Detection:** `-race` detector; memory profiling; leftover files after DELETE.

### Pitfall 10: Temp file cleanup and Windows file locking

**What goes wrong:** Extracted/edited images and result PDFs accumulate; `os.Remove` fails on Windows if a file is still open (pdfcpu or image decoders may hold handles); temp dirs are not removed on panic or early error; paths built from doc IDs allow traversal.

**Why it happens:** Cleanup is an afterthought; Windows locks open files.

**Consequences:** Disk fills; "file in use" errors on Windows; path traversal security issue.

**Prevention:**
- Create one `os.MkdirTemp("", "ocrpdf-")` per document and store its path; remove the whole dir on DELETE and via `defer` on error paths.
- Close all file handles (including pdfcpu readers) before removal; flush/close `*os.File` and image files.
- Use `defer os.RemoveAll(dir)` in handlers and tests; use `t.TempDir()` in tests.
- Generate server-side UUID doc IDs; validate with a regex; use `filepath.Clean` and a containment check against the storage root.
- Enforce max upload size with `http.MaxBytesReader` and validate PDF magic bytes before accepting.

**Detection:** Post-DELETE directory listing; Windows-specific CI run.

### Pitfall 11: Concurrency races on document edits

**What goes wrong:** Two concurrent `POST /edits` for the same document mutate the same page image and word list; `GET /result.pdf` rebuilds while an edit is in flight; the document's slice of page images is appended/read concurrently.

**Why it happens:** `net/http` runs each request in its own goroutine; the store only guards the map, not per-document state.

**Consequences:** Corrupted PNGs; lost edits; data races; partial PDFs.

**Prevention:**
- Add a per-document mutex; hold it for the duration of an edit or rebuild (not across the whole request if avoidable, but correctness first).
- Snapshot immutable copies of page/word data under the lock and process outside; write results back under the lock.
- Make rebuild read a stable snapshot of current image paths; invalidate/cache the result PDF on edit.
- Use `sync.RWMutex` per document for read-heavy preview serving.

**Detection:** `go test -race`; concurrent integration test with two edits on one doc.

### Pitfall 12: Large PDF handling (memory and latency)

**What goes wrong:** Upload OCRs every page synchronously before responding; all extracted/edited PNGs are held for the process lifetime; rebuild re-imports all images on every request; a huge PDF can OOM or time out at the gateway.

**Why it happens:** No background jobs, caching, or bounds.

**Consequences:** Request timeouts; OOM; poor throughput; gateway 502/504.

**Prevention:**
- Enforce max upload size, max page count, and max image dimensions; reject early with 413/400.
- Run OCR with `exec.CommandContext` and a timeout; kill on context cancellation.
- Cache the rebuilt PDF per document and invalidate on edit; stream the file with `http.ServeFile`/`io.Copy`.
- Spool images to disk (not memory) and delete on cleanup; add TTL eviction.
- Consider moving OCR to a background worker with a status endpoint for large docs; at minimum document the sync limitation.
- Cap Tesseract concurrency with a semaphore to avoid CPU thrash.

**Detection:** Load test with a 100-page scanned PDF; memory profiling.

## Moderate Pitfalls

### Pitfall: gosseract API name mismatch

**What goes wrong:** The plan references `GetTextBoxes`, but gosseract v2.4.1 exposes `GetBoundingBoxes(PageIteratorLevel)` and `GetBoundingBoxesVerbose()`. Implementing verbatim fails to compile.

**Prevention:** Use `GetBoundingBoxesVerbose()` and map `BoundingBox{Text, Confidence, Box}` to `Word{x0,y0,x1,y1}` (Box is a `*image.Rectangle` or similar; verify field names at implementation). Keep gosseract behind the build tag and add a compile-only CI job.

### Pitfall: Soft masks and multiple images per page

**What goes wrong:** pdfcpu may emit masks as separate PNGs; selecting by largest area usually picks the photo, but a mask with larger dimensions can win.

**Prevention:** Prefer images whose decoded format is RGB/RGBA with substantial non-mask content; filter by minimum area and aspect ratio; log candidates.

### Pitfall: Tesseract locale/decimal and empty text

**What goes wrong:** Confidence parsing assumes integer; Tesseract emits floats. Empty `text` field yields fewer CSV fields.

**Prevention:** ParseFloat; handle variable field counts; skip empty-text level-5 rows only if desired (keep them for layout? usually skip).

### Pitfall: Font file missing on Linux/macOS

**What goes wrong:** Default `C:\Windows\Fonts\arial.ttf` does not exist; `gg.LoadFontFace` errors.

**Prevention:** Resolve via `FONT_PATH`, fall back to a bundled font or a PATH search; fail fast at startup with a clear message.

### Pitfall: Result PDF page size vs edited image size

**What goes wrong:** If the edited PNG dimensions differ from the extracted image (e.g., re-encoded), rebuild scales to the configured PageDim and may resample.

**Prevention:** Keep edited images at the original pixel dimensions; do not re-encode with different size; preserve PNG.

### Pitfall: Race detector not run

**What goes wrong:** Concurrency bugs ship to production.

**Prevention:** `go test -race ./...` in CI on every PR.

## Minor Pitfalls

### Pitfall: Empty document ID or unknown doc

**Prevention:** Return 404 with a JSON error; never build paths from the raw ID without validation.

### Pitfall: Tesseract not on PATH

**Prevention:** Default to the installed path; verify the binary exists at startup; set `TESSDATA_PREFIX` programmatically.

### Pitfall: Case sensitivity of file extensions

**Prevention:** Accept `.pdf` case-insensitively; normalize extensions.

### Pitfall: Partial failure leaves temp dirs

**Prevention:** `defer os.RemoveAll` on every handler; cleanup on error.

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| pdfio extraction | Multiple images per page, generated filenames | Enumerate + pick largest by area; clean dir |
| pdfio rebuild | Appending to existing outFile; A4 default stretch | Remove first; set PageDim from source |
| OCR CLI | TSV header/empty lines/conf -1 | encoding/csv, filter level 5 |
| OCR gosseract | `GetTextBoxes` does not exist; CGO headers | Use `GetBoundingBoxesVerbose`; build tag |
| edit erase | Non-uniform backgrounds | Median ring sampling; configurable |
| edit render | Baseline at bbox bottom; height overflow | Metrics-based baseline; clamp height |
| api store | Unbounded memory; races | Per-doc mutex; TTL; -race |
| temp files | Windows locking; traversal | MkdirTemp per doc; UUID IDs |
| large PDFs | Sync OCR; OOM | Size/page caps; cache rebuild |
| rebuild | Stale result cache | Invalidate on edit |

## Sources

- pdfcpu v0.16.3 API docs: https://pkg.go.dev/github.com/pdfcpu/pdfcpu/pkg/api (ExtractImagesFile, ImportImagesFile, DefaultImportConfig, PageDimsFile) — HIGH
- pdfcpu Import struct source: https://raw.githubusercontent.com/pdfcpu/pdfcpu/v0.16.3/pkg/pdfcpu/importImage.go — HIGH
- Tesseract TSV format: https://tomrochette.com/tesseract-tsv-format and https://tesseract-ocr.github.io/tessdoc/Command-Line-Usage.html — HIGH
- Tesseract coordinate system: standard image origin top-left; PDF origin bottom-left — MEDIUM
- fogleman/gg API: https://github.com/fogleman/gg — MEDIUM
- gosseract Windows CGO notes: https://github.com/otiai10/gosseract/blob/main/knowledge/windows-support.md — MEDIUM
- Go sync/temp/concurrency: https://pkg.go.dev/sync, https://pkg.go.dev/os, https://go.dev/doc/articles/race_detector.html — HIGH
- Codebase concerns: `.planning/codebase/CONCERNS.md` — HIGH (internal)