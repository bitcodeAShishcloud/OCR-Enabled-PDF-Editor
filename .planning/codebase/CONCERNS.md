# Codebase Concerns

**Analysis Date:** 2026-10-27

## Tech Debt

**No implementation exists yet:**
- Issue: The repository contains only `go.mod`, `go.sum`, and a design plan (`.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`). There are no `cmd/`, `internal/`, test, or README files. The entire PDF OCR text-editing backend described in the plan is unbuilt.
- Files: `go.mod`, `go.sum`, `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`
- Impact: Nothing is runnable; every downstream phase (plan/execute/verify) is blocked on scaffolding.
- Fix approach: Scaffold the module layout from the plan, then implement package-by-package (`pdfio` -> `ocr` -> `edit` -> `api` + `main.go`).

**Dependencies declared but unused / mis-marked indirect:**
- Issue: All five requires in `go.mod` carry `// indirect`, yet the plan intends them as direct dependencies. `go mod tidy` (run with no source) removes the entire require block, confirming nothing imports them.
- Files: `go.mod`, `go.sum`
- Impact: Module state does not reflect intended dependency graph; a future `go mod tidy` could drop deps before code lands.
- Fix approach: Implement source that imports them, then run `go mod tidy` so Go reclassifies them as direct.

**Plan references a non-existent gosseract API:**
- Issue: The plan names the gosseract backend method `GetTextBoxes`, but `github.com/otiai10/gosseract/v2@v2.4.1` exposes `GetBoundingBoxes(PageIteratorLevel)` and `GetBoundingBoxesVerbose()` (no `GetTextBoxes`).
- Files: `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`
- Impact: The gosseract backend will not compile as written; rework required to map `BoundingBox` fields (Text/Confidence/Box) onto the internal `Word` type.
- Fix approach: Implement the gosseract backend using `GetBoundingBoxesVerbose()` and translate boxes to `x0,y0,x1,y1`.

## Known Bugs

**No runtime code exists, so no runtime bugs have been observed.** The following are design-level defects in the plan that will become bugs if implemented verbatim:

**pdfcpu extraction filename coupling:**
- Symptoms: `api.ExtractImagesFile(ctx, inFile, outDir, selectedPages, conf)` writes images to `outDir` with pdfcpu-generated names and returns only an error. The plan assumes one "largest image per page" can be selected, but pdfcpu may emit multiple images per page (soft masks, thumbnails, OCG layers).
- Files: `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`
- Trigger: Scanned PDFs whose embedded images include masks or multiple XObjects.
- Workaround: After extraction, enumerate `outDir`, group by page, and pick the largest by pixel area; clean `outDir` per document to avoid stale files.

**TSV column parsing (CLI backend):**
- Symptoms: Tesseract TSV has 12 columns and level-5 rows; naive `strings.Split` on tabs breaks on empty trailing fields and locale/whitespace variations. Confidence `-1` (non-text rows) must be filtered.
- Files: `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`
- Trigger: Pages with no detected words or Tesseract version output differences.
- Workaround: Use `encoding/csv` with tab delimiter and skip rows where level != 5 or conf < 0.

## Security Considerations

**Unauthenticated, in-memory document store:**
- Risk: No authentication/authorization; any client can upload, read, edit, or delete any document by guessing/leaking a doc ID. In-memory map is lost on restart and grows without bound.
- Files: `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md` (api/handlers.go)
- Current mitigation: Mutex-protected map (planned).
- Recommendations: Use unguessable UUID doc IDs, add an auth layer or scoped tokens, add TTL/eviction, and persist to disk or object storage.

**Arbitrary file upload (PDF):**
- Risk: Malicious/crafted PDFs can trigger heavy CPU/memory in pdfcpu or tesseract (DoS). No size limits or content-type validation planned.
- Files: `api/handlers.go` (planned)
- Current mitigation: None planned.
- Recommendations: Enforce max upload size, validate PDF magic bytes / pdfcpu parse before accepting, cap page count and image dimensions, and run OCR with timeouts.

**Subprocess invocation of Tesseract:**
- Risk: If image paths or arguments are built via shell strings, path injection is possible. Also `TESSDATA_PREFIX`/`TESSERACT_PATH` from environment can be tampered with.
- Files: `internal/ocr/cli.go` (planned)
- Current mitigation: Plan says to use full path from `TESSERACT_PATH`.
- Recommendations: Use `exec.Command` with discrete args (never `sh -c`), validate resolved binary path, and sanitize/quote any user-derived filename components.

**Path traversal in document IDs and file serving:**
- Risk: Doc IDs used to build filesystem paths (`outDir`, image paths) without validation could escape the storage root.
- Files: `internal/pdfio/*.go`, `api/handlers.go` (planned)
- Current mitigation: None planned.
- Recommendations: Generate server-side UUIDs for doc IDs; never accept user-supplied path segments; use `filepath.Clean` + root containment checks.

**Edited-image rebuild trusts client input:**
- Risk: `POST /edits` accepts `newText`, `font`, `size`, `color` without validation; oversized text or extreme sizes could panic the renderer or produce huge images.
- Files: `internal/edit/render.go`, `api/handlers.go` (planned)
- Current mitigation: Font size auto-fit (planned).
- Recommendations: Bound `newText` length, validate color hex, clamp font size, and cap output dimensions.

## Performance Bottlenecks

**Synchronous OCR of all pages on upload:**
- Problem: `POST /api/documents` OCRs every page before responding; large scanned PDFs will block the request and risk gateway timeouts.
- Files: `api/handlers.go` (planned)
- Cause: No background job or progress reporting.
- Improvement path: Move OCR to a background worker/queue, return a job ID, and expose status polling.

**Full PDF rebuild on every result request:**
- Problem: `GET /result.pdf` re-imports all current page images via pdfcpu each call.
- Files: `internal/pdfio/build.go` (planned)
- Cause: No caching of rebuilt PDF.
- Improvement path: Cache the rebuilt PDF per document and invalidate on edit; stream the file.

**In-memory retention of all page images:**
- Problem: Every uploaded document's extracted/edited PNGs are held for the process lifetime.
- Files: `api/handlers.go` (planned)
- Cause: No eviction or disk spooling.
- Improvement path: Spool images to a temp dir with TTL and delete on `DELETE`; add a memory/disk cap.

## Fragile Areas

**Background erase sampling:**
- Files: `internal/edit/erase.go` (planned)
- Why fragile: Filling the padded bbox with a color sampled from the surrounding pixel ring assumes a uniform background. Gradients, paper texture, shadows, or overlapping glyphs will leave visible boxes.
- Safe modification: Make padding/erase strategy configurable; fall back to median/edge-aware fill; add a "no erase" option.
- Test coverage: No tests planned for this heuristic.

**Text color sampling from darkest pixels:**
- Files: `internal/edit/render.go` (planned)
- Why fragile: Sampling the darkest pixels assumes dark text on light background. Fails for light/colored text or inverted scans; binary search font-fit may distort aspect ratio on unusual bboxes.
- Safe modification: Allow explicit color override (already in API) and default to black when sampling is unreliable; clamp fit ratio.
- Test coverage: Only font-fit unit test planned.

**Windows-hardcoded default paths:**
- Files: `.zcode/plans/plan-session_6bec9601b365e39dcd71ebb4a5c96fae.md`
- Why fragile: Defaults `C:\Windows\Fonts\arial.ttf` and `C:\Program Files\Tesseract-OCR\tesseract.exe` are Windows-only; the same binary cannot run on Linux/macOS without env overrides, and font availability differs.
- Safe modification: Resolve font via `FONT_PATH` or a bundled font; detect tesseract on PATH before falling back to the Windows default.

**gosseract backend unbuildable on this machine:**
- Files: build tag `gosseract`
- Why fragile: `gosseract/v2` requires CGO plus Tesseract development headers, which are not installed here (`tesseract_capi.h` not found). The gosseract backend cannot be compiled or tested in this environment without provisioning headers.
- Safe modification: Keep the CLI backend as the default and gate gosseract behind the build tag (as planned); document exact header installation steps per OS; add a CI job that builds with `-tags gosseract`.
- Test coverage: gosseract path untestable locally.

**pdfcpu API drift:**
- Files: `internal/pdfio/extract.go`, `internal/pdfio/build.go` (planned)
- Why fragile: The plan acknowledges pdfcpu signatures shift between versions; extraction returns files rather than in-memory objects, coupling code to on-disk layout.
- Safe modification: Wrap pdfcpu calls behind internal functions with integration tests; pin pdfcpu version in `go.mod`.

**Raster-flattened output:**
- Files: `internal/pdfio/build.go` (planned)
- Why fragile: Output PDF replaces each page with a raster image, losing vector text/searchability and inflating file size; acceptable for scanned PDFs but wrong for mixed content.
- Safe modification: Document this limitation (plan does); consider preserving original vector pages when no edit occurs.

## Scaling Limits

**In-memory document map:**
- Current capacity: Single process memory.
- Limit: OOM as concurrent uploads accumulate; no horizontal scaling because state is process-local.
- Scaling path: Move document state to shared storage (disk/S3) keyed by doc ID; make handlers stateless.

**Single Tesseract process per OCR call:**
- Current capacity: One invocation per page, serialized by the HTTP handler.
- Limit: Throughput ceiling on multi-page documents; CPU-bound.
- Scaling path: Worker pool with bounded concurrency; reuse a long-lived tesseract process if supported.

## Dependencies at Risk

**github.com/fogleman/gg v1.3.0:**
- Risk: Last release 2019; effectively unmaintained. API and font handling may have latent bugs.
- Impact: Rendering and font fitting depend on it; migration later is costly.
- Migration plan: Evaluate `golang.org/x/image/font` + `freetype` or a maintained fork; keep gg usage behind `internal/edit`.

**github.com/otiai10/gosseract/v2 v2.4.1:**
- Risk: Requires CGO and system Tesseract headers; headers absent on this machine; API naming in plan is wrong (`GetTextBoxes`).
- Impact: Optional backend cannot be built/tested here; divergence between CLI and gosseract behavior.
- Migration plan: Verify against `GetBoundingBoxesVerbose`; keep behind build tag; add CI.

**github.com/pdfcpu/pdfcpu v0.16.3:**
- Risk: API instability across versions; extraction writes to disk with generated names.
- Impact: Breakage on upgrade; fragile filename parsing.
- Migration plan: Pin version; wrap calls; add integration tests with fixture PDFs.

**Go toolchain 1.26.5:**
- Risk: Very recent (non-LTS) toolchain; contributors on older Go cannot build.
- Impact: Reproducibility and CI compatibility.
- Migration plan: Lower the `go` directive to the minimum actually required (likely 1.21+) after verifying APIs.

## Missing Critical Features

**No tests:**
- Problem: Zero test files exist; plan calls for TSV parsing and font-fit unit tests plus an end-to-end pipeline test, none implemented.
- Blocks: Verification of OCR/edit correctness; safe refactoring.

**No README / setup docs:**
- Problem: Setup (Tesseract install, font path, build tags) is only in the plan, not in-repo.
- Blocks: Onboarding and reproducible builds.

**No observability or lifecycle handling:**
- Problem: No logging, health endpoint, graceful shutdown, or request timeouts planned.
- Blocks: Production readiness and debugging.

**No input limits or rate limiting:**
- Problem: Upload/edit endpoints have no bounds.
- Blocks: Safe public deployment.

## Test Coverage Gaps

**Entire codebase untested:**
- What's not tested: All packages (none exist yet) — extraction, OCR parsing, erase/render, handlers, rebuild.
- Files: (no source files present)
- Risk: Any implementation lands without regression safety; the planned unit tests (TSV parsing, font-fit) and E2E pipeline test are not yet written.
- Priority: High

**gosseract backend path:**
- What's not tested: Cannot be compiled in this environment due to missing Tesseract headers.
- Files: build tag `gosseract`
- Risk: Silent divergence between OCR backends.
- Priority: Medium

---

*Concerns audit: 2026-10-27*
