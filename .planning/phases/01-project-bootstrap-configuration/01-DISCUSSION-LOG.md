# Phase 1: Project Bootstrap & Configuration - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-27
**Phase:** 01-project-bootstrap-configuration
**Areas discussed:** HTTP Server, Configuration, OCR Abstraction, Logging, Module/Build

---

## HTTP Server

| Option | Description | Selected |
|--------|-------------|----------|
| stdlib net/http + ServeMux | Go 1.22+ method patterns, no third-party router | ✓ |
| chi / gin / echo | Third-party router | |

**User's choice:** [auto] stdlib net/http + ServeMux (recommended default)
**Notes:** Auto mode selected recommended default.

---

## Configuration

| Option | Description | Selected |
|--------|-------------|----------|
| Env vars with defaults, fail fast on invalid | Typed Config struct at startup | ✓ |
| Config file | YAML/TOML file | |

**User's choice:** [auto] Env vars with defaults (recommended default)
**Notes:** Auto mode selected recommended default.

---

## OCR Abstraction

| Option | Description | Selected |
|--------|-------------|----------|
| Engine interface + factory by OCR_BACKEND | CLI default, gosseract behind build tag | ✓ |

**User's choice:** [auto] (recommended default from plan)
**Notes:** Auto mode selected recommended default.

---

## Logging

| Option | Description | Selected |
|--------|-------------|----------|
| log/slog JSON to stdout | Standard library | ✓ |

**User's choice:** [auto] (recommended default)
**Notes:** Auto mode selected recommended default.

---

## Module/Build

| Option | Description | Selected |
|--------|-------------|----------|
| CGO-free default build | gosseract behind build tag | ✓ |

**User's choice:** [auto] (recommended default from plan)
**Notes:** Auto mode selected recommended default.

## the agent's Discretion

- Exact file layout within cmd/server (main.go only).
- Error message wording.
- Whether to add a Makefile — deferred.

## Deferred Ideas

- Makefile / build scripts — use go tooling directly for v1.
- Structured request logging middleware — basic slog sufficient for v1.