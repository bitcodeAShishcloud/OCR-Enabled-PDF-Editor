---
phase: 05-edit-rebuild-end-to-end-verification
plan: 01
subsystem: api
tags: [go, edit, erase, render, font-fit, color-sampling, pdf, ocr, e2e]
requires:
  - phase: 04-document-store-upload-preview-cleanup-api
    provides: document store, upload/OCR, GetPage, DeleteDoc handlers
provides:
  - internal/edit package (ToRGBA, ParseColor, SampleBackground, SampleTextColor, Erase, FitFontSize, RenderText)
  - per-document mutex and mutable current image path
  - POST /api/documents/{id}/edits handler
  - GET /api/documents/{id}/result.pdf handler
  - font-fit unit test and synthetic E2E round-trip test
affects: [ship, verify-work]
actuals:
  tokens: 7112
  tasks: 3
  commits: 3
  plan_head_before: ca6bfb894267567c4f49c6e9ea5608a9af624f22
  plan_head_after: d1e6b23eb51933837fbc98ce37f914f567e7681f
tech-stack:
  added: []
  patterns:
    - ring-sampled background erase with median per-channel sampling
    - binary-search font fit on width and height
    - darkest-pixel text color sampling
    - metrics-based baseline placement
    - per-document mutex with store->doc lock ordering
key-files:
  created:
    - internal/edit/edit.go
    - internal/edit/erase.go
    - internal/edit/color.go
    - internal/edit/render.go
    - internal/edit/edit_test.go
    - internal/api/e2e_edit_test.go
  modified:
    - internal/api/store.go
    - internal/api/handlers.go
    - cmd/server/main.go
key-decisions:
  - "Background sampled from a 6px ring with dark pixels excluded; median per channel."
  - "Font size auto-fitted by binary search over [4, min(boxH,200)] on both width and height."
  - "Text color sampled from bbox pixels darker than bgLum-30, fallback to darkest 10%, then black."
  - "Baseline = bbox.Max.Y - Descent/64, clamped to bbox.Min.Y + Ascent/64."
  - "Edited page image overwrites Page.ImagePath; result rebuilds from current paths."
requirements-completed: [EDIT-01, EDIT-02, EDIT-03, EDIT-04, EDIT-05, EDIT-06, API-03, API-04, TEST-02, TEST-03]
patterns-established:
  - "Pure raster edit functions over *image.RGBA; handlers only orchestrate."
  - "Per-doc sync.Mutex; store mutex -> doc mutex ordering."
duration: 5min
completed: 2026-10-28
status: complete
---

# Phase 5: Edit, Rebuild & End-to-End Verification Summary

**Ring-sampled background erase, binary-search font fit, darkest-pixel text color, metrics-based baseline, edit and rebuild endpoints, plus font-fit and synthetic E2E tests — all passing under -race with CGO-free build.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-10-28T00:00:00Z
- **Completed:** 2026-10-28T00:05:00Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- Implemented `internal/edit` package with ToRGBA, ParseColor, SampleBackground, SampleTextColor, Erase, FitFontSize, RenderText.
- Added per-document mutex and mutable current image path; updated GetPage locking.
- Implemented `POST /api/documents/{id}/edits` returning edited preview PNG.
- Implemented `GET /api/documents/{id}/result.pdf` rebuilding and streaming final PDF.
- Wired `FontPath` and both routes in `cmd/server/main.go`.
- Added `TestFitFontSize` and `TestEditTracer` (internal/edit) and `TestEditRebuildRoundTrip` (internal/api).

## Task Commits

1. **Task 1: internal/edit package** - `983476e` (feat)
2. **Task 2: store mutex + Edit/Result handlers** - `bea1887` (feat)
3. **Task 3: E2E round-trip test** - `d1e6b23` (test)

## Files Created/Modified

- `internal/edit/edit.go` - ToRGBA and ParseColor helpers
- `internal/edit/erase.go` - Erase with padded fill
- `internal/edit/color.go` - luminance, SampleBackground, SampleTextColor, median helpers
- `internal/edit/render.go` - FitFontSize and RenderText with metrics baseline
- `internal/edit/edit_test.go` - FitFontSize unit test and tracer pipeline test
- `internal/api/store.go` - added sync.Mutex to Document
- `internal/api/handlers.go` - FontPath, Edit, Result handlers, GetPage locking
- `internal/api/e2e_edit_test.go` - synthetic end-to-end round trip
- `cmd/server/main.go` - FontPath wiring and two new routes

## Decisions Made

- Preview response returns raw edited PNG bytes.
- wordIds are zero-based indices into Page.Words.
- Editing overwrites Page.ImagePath with edited PNG path.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Project complete and verified. All tests pass under -race with CGO_ENABLED=0.

---
*Phase: 05-edit-rebuild-end-to-end-verification*
*Completed: 2026-10-28*
## Self-Check: PASSED
