# OCR PDF Text-Editing Backend (Go)

HTTP backend that takes a scanned PDF, OCRs per-page raster images for word-level text plus bounding boxes, replaces a detected name with new text drawn in place, and rebuilds the final PDF. Module `ocrpdf`.

## Prerequisites

- Go 1.26.5
- Tesseract OCR 5.5.0 (Windows default path `C:\Program Files\Tesseract-OCR\tesseract.exe`)
- Arial TTF (`C:\Windows\Fonts\arial.ttf`)

## Environment variables

| Var | Default | Description |
|---|---|---|
| `TESSERACT_PATH` | `C:\Program Files\Tesseract-OCR\tesseract.exe` | Path to the tesseract binary. |
| `FONT_PATH` | `C:\Windows\Fonts\arial.ttf` | Path to a TTF font. |
| `OCR_BACKEND` | `cli` | `cli` or `gosseract`. |
| `PORT` | `8080` | Listen port (1-65535). |

## Run

```
go run ./cmd/server
```

Health check:

```
GET http://localhost:8080/healthz  -> {"status":"ok"}
```

## Builds

Default build is CGO-free:

```
CGO_ENABLED=0 go build ./...
```

Tagged build enables the gosseract backend (requires Tesseract/Leptonica dev headers):

```
go build -tags gosseract ./...
```

## Shutdown

The server shuts down gracefully on SIGINT/SIGTERM with a 10s timeout.
