---
phase: 04-document-store-upload-preview-cleanup-api
plan: 01
subsystem: api
tags: [http, multipart, pdf, ocr, store, uuid, race]

requires:
  - phase: 01-project-bootstrap
    provides: cmd/server skeleton and healthz handler
  - phase: 02-pdf-i-o-layer
    provides: pdfio.ExtractImages and pdfapi.PageCountFile
  - phase: 03-ocr-engine-tsv-parsing
    provides: ocr.Engine and ocr.Word

provides:
  - internal/api package with mutex-protected Store, Page, Document types
  - UUID v4 generation via crypto/rand (newUUID)
  - POST /api/documents upload with magic-byte/size/page validation and OCR
  - GET /api/documents/{id}/pages/{n} PNG preview
  - DELETE /api/documents/{id} cleanup
  - cmd/server wiring of Store + Handler and routes

affects: [phase-05-edit-rebuild]

tech-stack:
  added: []
  patterns: [mutex-guarded in-memory map, http.MaxBytesReader size cap, success-flag deferred cleanup, DTO separation from domain types]

key-files:
  created:
    - internal/api/store.go
    - internal/api/ids.go
    - internal/api/handlers.go
    - internal/api/handlers_test.go
  modified:
    - cmd/server/main.go

key-decisions:
  - "Exported Upload/GetPage/DeleteDoc wrappers added alongside unexported methods so cross-package main.go and tests can call them"
  - "DELETE returns 204 No Content; preview returns 200 with Content-Type image/png"
  - "Empty word/page slices initialized with make(...,0,...) so JSON serializes as [] not null"

patterns-established:
  - "Lock only for map/pointer access; never during OCR/extraction/cleanup I/O"
  - "Per-document temp dir with success flag and deferred os.RemoveAll on failure"

requirements-completed: [API-01, API-02, API-05, API-06, API-07, TEST-04]

actuals:
  tokens: 4745
  tasks: 3
  commits: 1
  plan_head_before: d33358cefee1e6f9c182371220832b6d80bc8f94
  plan_head_after: dfddbf332e0e541822aaf844af83163166bfee94

coverage:
  - id: D1
    description: "POST /api/documents validates PDF and returns docId plus per-page words with bboxes"
    requirement: API-01
    verification:
      - kind: integration
        ref: "internal/api/handlers_test.go#TestUploadTracer"
        status: pass
    human_judgment: false
  - id: D2
    description: "GET /api/documents/{id}/pages/{n} serves PNG with image/png content type"
    requirement: API-02
    verification:
      - kind: integration
        ref: "internal/api/handlers_test.go#TestPreviewReturnsPNG"
        status: pass
    human_judgment: false
  - id: D3
    description: "DELETE /api/documents/{id} removes document and artifacts; later access 404"
    requirement: API-05
    verification:
      - kind: integration
        ref: "internal/api/handlers_test.go#TestDeleteRemovesDocument"
        status: pass
    human_judgment: false
  - id: D4
    description: "In-memory store is mutex-protected with no data races"
    requirement: API-06
    verification:
      - kind: unit
        ref: "internal/api/handlers_test.go#TestStoreConcurrent"
        status: pass
    human_judgment: false
  - id: D5
    description: "Document IDs are RFC 4122 v4 via crypto/rand"
    requirement: API-07
    verification:
      - kind: unit
        ref: "internal/api/handlers_test.go#TestUUIDFormat"
        status: pass
    human_judgment: false

metrics:
  duration: 25min
  completed: 2026-10-28
status: complete
---

# Phase 4 Plan 01: Document Store & Upload/Preview/Cleanup API Summary

**Mutex-protected in-memory document store keyed by crypto/rand UUID v4, with multipart PDF upload (magic-byte/size/page validation, pdfcpu extraction, per-page OCR), PNG page preview, and delete cleanup, all passing under -race**

## Performance

- **Duration:** 25 min
- **Started:** 2026-10-28T00:00:00Z
- **Completed:** 2026-10-28T00:25:00Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Implemented `Store`, `Document`, `Page` with mutex-guarded `Add`/`Get`/`Delete` (lock only for map access).
- Implemented `newUUID` (RFC 4122 v4 via crypto/rand).
- Implemented `upload`/`getPage`/`deleteDoc` handlers with validation caps and deferred cleanup.
- Wired Store + Handler and routes into `cmd/server/main.go`.
- Added comprehensive race-detector tests (9 tests) covering upload, preview, delete, validation, empty-slice serialization, concurrency, and UUID format.

## Task Commits

1. **Task 1-3: implementation + tests** - `dfddbf3` (feat)

## Files Created/Modified

- `internal/api/store.go` - Store, Document, Page types and mutex-protected methods
- `internal/api/ids.go` - newUUID via crypto/rand
- `internal/api/handlers.go` - Handler, DTOs, upload/getPage/deleteDoc
- `internal/api/handlers_test.go` - race-detector and HTTP tests
- `cmd/server/main.go` - wiring of Store + Handler and route registration

## Decisions Made

- Exported `Upload`/`GetPage`/`DeleteDoc` wrappers were added because unexported methods are inaccessible across packages; both forms are available.
- DELETE returns 204 No Content.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Exported handler method wrappers**
- **Found during:** Task 1 (wiring)
- **Issue:** Plan referenced unexported `h.upload` etc. from `cmd/server` (different package), which does not compile.
- **Fix:** Added exported wrappers `Upload`/`GetPage`/`DeleteDoc` delegating to unexported implementations.
- **Files modified:** `internal/api/handlers.go`, `cmd/server/main.go`
- **Verification:** `go build ./...` passes
- **Committed in:** dfddbf3

**Total deviations:** 1 auto-fixed (blocking)
**Impact on plan:** Required for compilation; no scope creep.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/api.Store.Get` available for Phase 5 to read page images and words.
- No blockers.

---
*Phase: 04-document-store-upload-preview-cleanup-api*
*Completed: 2026-10-28*
## Self-Check: PASSED