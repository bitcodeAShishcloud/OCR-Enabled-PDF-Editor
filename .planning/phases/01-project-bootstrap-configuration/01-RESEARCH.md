# Phase 1: Project Bootstrap & Configuration - Research

**Researched:** 2026-10-27
**Domain:** Go stdlib HTTP server bootstrap, env config, OCR engine abstraction, build tags
**Confidence:** HIGH

## Summary

Phase 1 delivers a runnable, CGO-free Go HTTP server (`cmd/server/main.go`) that reads environment-driven configuration, selects an OCR backend through a factory, exposes a health endpoint, and shuts down gracefully. No external (non-stdlib) dependencies are required for this phase; the OCR interface and gosseract stub are the only cross-package contracts introduced. All patterns are standard library (`net/http`, `log/slog`, `os/signal`, `context`, `strconv`, `os`), verified against the installed Go 1.26.5 toolchain.

**Primary recommendation:** Implement `main.go` with a typed `Config` struct parsed from env vars, an `http.ServeMux` using Go 1.22+ method patterns (`GET /healthz`), `slog.New(slog.NewJSONHandler(os.Stdout, nil))`, `signal.NotifyContext` for SIGINT/SIGTERM, and an `http.Server` with `Shutdown(ctx)` under a 10s timeout. Define `ocr.Engine`/`Word` in `internal/ocr/ocr.go` and a `NewEngine(cfg) (Engine, error)` factory in `internal/ocr/factory.go` (or `cli.go`), with the gosseract file guarded by `//go:build gosseract`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Use Go standard library `net/http` with `http.ServeMux` (Go 1.22+ method patterns). No third-party router. Reversible.
- **D-02:** Listen address from `PORT` env var, default `8080`; bind `0.0.0.0` (all interfaces). Fail fast on invalid port. Reversible.
- **D-03:** Graceful shutdown on SIGINT/SIGTERM with a 10s shutdown timeout. Reversible.
- **D-04:** `GET /healthz` returns 200 OK JSON `{status:"ok"}`. Reversible.
- **D-05:** Parse env vars at startup into a typed `Config` struct; missing optional vars use documented defaults. Reversible.
- **D-06:** `TESSERACT_PATH` defaults to `C:\Program Files\Tesseract-OCR\tesseract.exe`; `FONT_PATH` defaults to `C:\Windows\Fonts\arial.ttf`; `OCR_BACKEND` defaults to `cli`. Reversible.
- **D-07:** Invalid `OCR_BACKEND` value fails startup with a clear error. Reversible.
- **D-08:** Define `ocr.Engine` interface in `internal/ocr/ocr.go`: `Recognize(imagePath string) ([]Word, error)`, with `Word{Text string, Confidence float64, X0,Y0,X1,Y1 int}`. Reversible.
- **D-09:** `internal/ocr/cli.go` default backend invokes tesseract binary; `internal/ocr/gosseract.go` behind `//go:build gosseract`. Reversible.
- **D-10:** Backend selection via factory `NewEngine(cfg) (Engine, error)` reading `OCR_BACKEND`. Reversible.
- **D-11:** Use `log/slog` with JSON handler writing to stdout. Reversible.
- **D-12:** Module path `ocrpdf` (already in go.mod). Default build CGO-free. Reversible.
- **D-13:** Keep config in `cmd/server` main.go for now; extract if needed. Reversible.

### the agent's Discretion
- Exact file layout within `cmd/server` (main.go only).
- Error message wording.
- Whether to add a Makefile — deferred; use `go build`/`go test` directly.

### Deferred Ideas (OUT OF SCOPE)
- Makefile / build scripts — use go tooling directly for v1.
- Structured request logging middleware — basic slog sufficient for v1.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CONF-01 | Configuration via environment variables (`TESSERACT_PATH`, `FONT_PATH`, `OCR_BACKEND`) | `Config` struct populated by `os.Getenv` with documented defaults (D-05/D-06); `PORT` added for listen address (D-02). |
| CONF-02 | Default build is CGO-free | `//go:build gosseract` guard on `internal/ocr/gosseract.go`; default `go build ./...` with `CGO_ENABLED=0` succeeds. Verified by `go build` smoke test. |
| OCR-01 | OCR engine interface exposes `Recognize(imagePath) ([]Word, error)` with word text, confidence, and bounding box | `internal/ocr/ocr.go` defines `Engine` interface and `Word{Text, Confidence, X0, Y0, X1, Y1}` exactly per D-08. |
| OCR-06 | OCR backend selectable at startup via `OCR_BACKEND=cli\|gosseract` | `NewEngine(cfg) (Engine, error)` validates value and returns clear error on invalid (D-07/D-10). |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| HTTP listener / routing | API / Backend | — | Single Go process owns the listener; stdlib `net/http`. |
| Env config parsing | API / Backend | — | Startup-time, process-local; lives in `cmd/server/main.go` per D-13. |
| OCR backend selection | API / Backend | — | Factory resolves implementation at startup; consumed by handlers later. |
| Health endpoint | API / Backend | — | Liveness probe served by the same process. |
| Graceful shutdown | API / Backend | OS signals | `os/signal.NotifyContext` bridges OS signals to context cancellation. |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library `net/http` | Go 1.26.5 | HTTP server, routing via `ServeMux` | Locked by D-01; Go 1.22+ method patterns avoid third-party router. |
| Go standard library `log/slog` | Go 1.26.5 | Structured JSON logging to stdout | Locked by D-11; stdlib, no dependency. |
| Go standard library `os/signal` + `context` | Go 1.26.5 | Signal handling + cancellation | `NotifyContext` is the idiomatic graceful-shutdown primitive. |
| Go standard library `strconv` | Go 1.26.5 | Port string → int parsing | `strconv.Atoi` with fail-fast on invalid. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os` | Go 1.26.5 | Env var reads (`os.Getenv`), file existence checks | Config defaults and validation. |
| `time` | Go 1.26.5 | 10s shutdown timeout (`10 * time.Second`) | `context.WithTimeout` for `Server.Shutdown`. |
| `encoding/json` | Go 1.26.5 | Health endpoint JSON response | `json.NewEncoder(w).Encode(map...)`. |

**Installation:** No new modules. `go.mod` already declares `module ocrpdf` / `go 1.26.5`. No `go get` required for Phase 1.

## Package Legitimacy Audit

No external packages are installed in Phase 1 — only the Go standard library is used. `go.mod`/`go.sum` are untouched. **Packages removed due to [SLOP] verdict:** none. **Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### Recommended Project Structure (Phase 1 subset)
```
cmd/
└── server/
    └── main.go          # Config, server bootstrap, routing, graceful shutdown
internal/
└── ocr/
    ├── ocr.go           # Engine interface + Word type
    ├── cli.go           # default CLI backend (stub ok in Phase 1; full in Phase 3)
    └── gosseract.go     # //go:build gosseract stub
```
Note: `internal/ocr/cli.go` and `gosseract.go` may be minimal stubs in Phase 1 (interface satisfaction only) since OCR parsing is Phase 3. The factory must compile for both build configurations.

### Pattern 1: Env config into typed struct
**What:** Read env vars once at startup into `Config`; validate; fail fast.
**When to use:** Process bootstrap.
**Example:**
```go
// Source: Go stdlib os/signal, net/http (verified via go doc)
type Config struct {
    Port           int
    TesseractPath  string
    FontPath       string
    OCRBackend     string
}

func loadConfig() (Config, error) {
    cfg := Config{
        TesseractPath: `C:\Program Files\Tesseract-OCR\tesseract.exe`,
        FontPath:      `C:\Windows\Fonts\arial.ttf`,
        OCRBackend:    "cli",
    }
    if v := os.Getenv("TESSERACT_PATH"); v != "" { cfg.TesseractPath = v }
    if v := os.Getenv("FONT_PATH"); v != "" { cfg.FontPath = v }
    if v := os.Getenv("OCR_BACKEND"); v != "" { cfg.OCRBackend = v }
    // PORT
    portStr := os.Getenv("PORT")
    if portStr == "" { portStr = "8080" }
    p, err := strconv.Atoi(portStr)
    if err != nil || p <= 0 || p > 65535 { return Config{}, fmt.Errorf("invalid PORT %q: %w", portStr, err) }
    cfg.Port = p
    // validate OCR_BACKEND
    switch cfg.OCRBackend {
    case "cli", "gosseract":
    default:
        return Config{}, fmt.Errorf("invalid OCR_BACKEND %q (want cli|gosseract)", cfg.OCRBackend)
    }
    return cfg, nil
}
```

### Pattern 2: ServeMux with Go 1.22+ method patterns
**What:** `mux.HandleFunc("GET /healthz", healthHandler)` — method + path in one pattern.
**When to use:** All route registration.
**Example:**
```go
mux := http.NewServeMux()
mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
})
```
Verified: `go doc net/http.ServeMux` confirms `"GET /static/"` style method patterns are supported in this toolchain. [VERIFIED: go doc net/http.ServeMux]

### Pattern 3: slog JSON handler to stdout
**What:** `slog.New(slog.NewJSONHandler(os.Stdout, nil))` and set as default.
**Example:**
```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
slog.SetDefault(logger)
```
Verified: `go doc log/slog.NewJSONHandler`. [VERIFIED: go doc log/slog.NewJSONHandler]

### Pattern 4: Graceful shutdown with signal.NotifyContext
**What:** Listen for SIGINT/SIGTERM; on signal, call `srv.Shutdown(ctx)` with 10s timeout.
**Example:**
```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

srv := &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", cfg.Port), Handler: mux}

go func() {
    slog.Info("server listening", "addr", srv.Addr)
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        slog.Error("listen error", "err", err)
        os.Exit(1)
    }
}()

<-ctx.Done()
slog.Info("shutting down")
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := srv.Shutdown(shutdownCtx); err != nil {
    slog.Error("shutdown error", "err", err)
}
```
Verified: `go doc os/signal.NotifyContext` and `go doc net/http.Server.Shutdown`. [VERIFIED: go doc os/signal.NotifyContext, go doc net/http.Server.Shutdown]

Note: `syscall.SIGTERM` is available on Windows via `golang.org/x/sys`? Actually `syscall.SIGTERM` exists on Windows in the standard `syscall` package as a defined constant (os.Interrupt is the portable one). To be safe and portable, prefer `os.Interrupt` only, or use `syscall.SIGTERM` guarded. On Windows, `syscall.SIGTERM` is defined. [ASSUMED] — verify at implementation; alternative is to register only `os.Interrupt` on Windows and `syscall.SIGTERM` behind build tags. Simpler: use `os.Interrupt` and `syscall.SIGTERM` (both compile on Windows per stdlib). Recommend verifying with `go build` in Phase execution.

### Pattern 5: OCR Engine interface + factory
**What:** Interface in `internal/ocr/ocr.go`; factory `NewEngine(cfg) (Engine, error)`.
**Example:**
```go
// internal/ocr/ocr.go
package ocr

type Word struct {
    Text        string
    Confidence  float64
    X0, Y0, X1, Y1 int
}

type Engine interface {
    Recognize(imagePath string) ([]Word, error)
}
```
```go
// internal/ocr/factory (or cli.go)
func NewEngine(cfg config-ish) (Engine, error) {
    switch backend {
    case "cli":
        return &cliEngine{tesseractPath: ...}, nil
    case "gosseract":
        return newGosseractEngine() // only defined under build tag
    default:
        return nil, fmt.Errorf("invalid OCR_BACKEND %q", backend)
    }
}
```
Build-tag guard:
```go
//go:build gosseract

package ocr

// gosseractEngine implements Engine via gosseract/v2
```
And a non-tagged stub file:
```go
//go:build !gosseract

package ocr

func newGosseractEngine() (Engine, error) {
    return nil, errors.New("gosseract backend requires building with -tags gosseract")
}
```
This ensures default build compiles and selecting `gosseract` at runtime (without the tag) fails with a clear error. [ASSUMED] — exact factory placement (single file vs. split) is the agent's discretion per D-10.

### Anti-Patterns to Avoid
- **Don't import gosseract in default build** — always behind `//go:build gosseract`; otherwise CGO headers required and CONF-02 fails.
- **Don't use third-party router** (chi/gorilla) — locked D-01; stdlib ServeMux suffices for Phase 1 routes.
- **Don't call `http.ListenAndServe` without a Server instance** if you need graceful shutdown — use `&http.Server{}` + `Shutdown`.
- **Don't parse PORT with `os.LookupEnv` ignoring errors** — fail fast on invalid (D-02).
- **Don't log with `log.Printf`** — use `slog` JSON handler (D-11).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Signal handling | Custom signal channel loops | `os/signal.NotifyContext` | Handles context propagation + cleanup. |
| Graceful shutdown | Manual connection draining | `http.Server.Shutdown(ctx)` | Stdlib implements idle-connection wait + listener close. |
| JSON encoding | Manual string concat | `encoding/json` | Escaping correctness. |
| Port validation | Regex | `strconv.Atoi` + range check | Simpler, idiomatic. |

## Common Pitfalls

### Pitfall 1: ServeMux pattern method requires Go 1.22
**What goes wrong:** On Go < 1.22, `"GET /healthz"` is treated as a literal path and never matches.
**Why it happens:** Method patterns were added in Go 1.22.
**How to avoid:** Toolchain is Go 1.26.5 (verified `go version`). No risk here. [VERIFIED: go version go1.26.5]
**Warning signs:** 404 on `/healthz` for GET.

### Pitfall 2: Default build pulling CGO via gosseract
**What goes wrong:** `go build ./...` fails on Windows without Tesseract dev headers.
**How to avoid:** Guard `internal/ocr/gosseract.go` with `//go:build gosseract`; keep all gosseract imports inside that file. Verify with `CGO_ENABLED=0 go build ./...`.

### Pitfall 3: `os.Exit` inside handler goroutine
**What goes wrong:** Skips deferred cleanup.
**How to avoid:** Only `os.Exit` after logging fatal startup errors before serving; use `return` in main after shutdown.

### Pitfall 4: Shutdown context too short
**What goes wrong:** Active requests aborted.
**How to avoid:** 10s timeout per D-03; document that long-running requests may be cut.

### Pitfall 5: Windows path defaults with backslashes in raw strings
**What goes wrong:** `\T` etc. interpreted as escape in double-quoted strings.
**How to avoid:** Use raw string literals (backticks) for default Windows paths.

## Code Examples

### Health handler
```go
// Source: Go stdlib net/http (verified via go doc)
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

### main skeleton
```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "syscall"
    "time"

    "ocrpdf/internal/ocr"
)

func main() {
    cfg, err := loadConfig()
    if err != nil {
        fmt.Fprintln(os.Stderr, "config error:", err)
        os.Exit(1)
    }

    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    engine, err := ocr.NewEngine(cfg) // cfg carries backend + tesseract path
    if err != nil {
        slog.Error("ocr backend init failed", "err", err)
        os.Exit(1)
    }
    _ = engine

    mux := http.NewServeMux()
    mux.HandleFunc("GET /healthz", healthHandler)

    srv := &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", cfg.Port), Handler: mux}

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    go func() {
        slog.Info("server listening", "addr", srv.Addr, "backend", cfg.OCRBackend)
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            slog.Error("listen error", "err", err)
            os.Exit(1)
        }
    }()

    <-ctx.Done()
    slog.Info("shutdown signal received")
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := srv.Shutdown(shutdownCtx); err != nil {
        slog.Error("graceful shutdown failed", "err", err)
        os.Exit(1)
    }
    slog.Info("server stopped")
}
```

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build/run | ✓ | 1.26.5 | — |
| Tesseract binary | CLI OCR backend (Phase 3) | ✓ | 5.5.0 at `C:\Program Files\Tesseract-OCR\tesseract.exe` | `TESSERACT_PATH` override |
| Font file (Arial TTF) | Edit rendering (Phase 5) | ✓ | `C:\Windows\Fonts\arial.ttf` | `FONT_PATH` override |

**Missing dependencies with no fallback:** none for Phase 1 (server itself has no external deps).
**Missing dependencies with fallback:** none.

Phase 1 does not require validating Tesseract/font existence at startup? Success criteria says "validates that Tesseract and the font exist." Recommend `os.Stat` checks in `loadConfig` or a `validatePaths` step for `TESSERACT_PATH` and `FONT_PATH`, failing fast with clear messages. [ASSUMED] — D-05/D-06 do not explicitly require existence validation, but ROADMAP success criterion 2 implies it. Recommend including it.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `http.DefaultServeMux` + `http.Handle` | `http.NewServeMux()` + method patterns | Go 1.22 | Method-aware routing without third-party router. |
| `log` package | `log/slog` | Go 1.21 | Structured JSON logging stdlib. |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `syscall.SIGTERM` compiles on Windows in stdlib `syscall` package. | Code Examples / Pattern 4 | Build failure on Windows; mitigate by verifying `go build` and falling back to `os.Interrupt` only. |
| A2 | Factory split into tag-guarded stub files (`newGosseractEngine`) is the cleanest approach. | Pattern 5 | Alternative: single file with build-tagged function; planner decides file layout. |
| A3 | Path existence validation (`os.Stat`) for Tesseract/font is in scope for Phase 1. | Environment Availability | If not required, remove validation step; low risk. |
| A4 | Config passed to `ocr.NewEngine` (vs. backend string + paths). | Code Examples | Signature detail; planner can refine. |

## Open Questions

1. **Config location for OCR factory.** Should `NewEngine` take the whole `Config` or just `OCRBackend` + `TesseractPath`? Recommendation: accept a small `ocr.Config` or individual params to avoid import cycle (`internal/ocr` must not import `cmd/server`). Define `ocr.Config{Backend, TesseractPath}` in `internal/ocr/ocr.go`. [ASSUMED]
2. **Health response exact shape.** D-04 says `{status:"ok"}`. Use `map[string]string{"status":"ok"}` → JSON keys sorted? `encoding/json` marshals map keys sorted, output `{"status":"ok"}`. Acceptable.

## Security Domain

Security enforcement enabled (ASVS L1).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | No auth in v1 (out of scope). |
| V3 Session Management | no | Stateless. |
| V4 Access Control | no | No user accounts. |
| V5 Input Validation | yes | `strconv.Atoi` + range check for PORT; enum validation for `OCR_BACKEND`. |
| V6 Cryptography | no | No crypto in this phase. |

### Known Threat Patterns for Go stdlib HTTP

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Invalid port / bind misuse | Denial of Service | Validate PORT range; bind configured address only. |
| Unvalidated env config | Tampering | Enum validation for `OCR_BACKEND`; path checks. |
| Resource exhaustion (no timeouts) | DoS | Set `ReadHeaderTimeout`/`IdleTimeout` on `http.Server` (recommended). |

## Sources

### Primary (HIGH confidence)
- `go doc net/http.ServeMux` — method patterns, Go 1.26.5 toolchain. [VERIFIED]
- `go doc log/slog.NewJSONHandler` — JSON handler. [VERIFIED]
- `go doc os/signal.NotifyContext` — signal context. [VERIFIED]
- `go doc net/http.Server.Shutdown` — graceful shutdown. [VERIFIED]
- `go version` → go1.26.5 windows/amd64. [VERIFIED]

### Secondary (MEDIUM confidence)
- Project docs: `.planning/CONTEXT.md`, `REQUIREMENTS.md`, `ROADMAP.md`, `.planning/codebase/*`. [CITED]

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all stdlib, verified via go doc.
- Architecture: HIGH — locked decisions D-01..D-13.
- Pitfalls: MEDIUM — Windows syscall.SIGTERM assumption flagged.

**Research date:** 2026-10-27
**Valid until:** 2026-11-26 (30 days; stdlib stable)
