package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("TESSERACT_PATH", "")
	t.Setenv("FONT_PATH", "")
	t.Setenv("PORT", "")
	t.Setenv("OCR_BACKEND", "")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.OCRBackend != "cli" {
		t.Errorf("OCRBackend = %q, want cli", cfg.OCRBackend)
	}
	if cfg.TesseractPath != `C:\Program Files\Tesseract-OCR\tesseract.exe` {
		t.Errorf("TesseractPath = %q, want default", cfg.TesseractPath)
	}
	if cfg.FontPath != `C:\Windows\Fonts\arial.ttf` {
		t.Errorf("FontPath = %q, want default", cfg.FontPath)
	}
}

func TestLoadConfigInvalidPort(t *testing.T) {
	t.Setenv("PORT", "abc")
	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for invalid PORT")
	}
	if !strings.Contains(err.Error(), "PORT") {
		t.Errorf("error %q does not mention PORT", err.Error())
	}
}

func TestLoadConfigInvalidBackend(t *testing.T) {
	t.Setenv("OCR_BACKEND", "nope")
	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for invalid OCR_BACKEND")
	}
	if !strings.Contains(err.Error(), "OCR_BACKEND") {
		t.Errorf("error %q does not mention OCR_BACKEND", err.Error())
	}
}

func TestLoadConfigInvalidPath(t *testing.T) {
	t.Setenv("TESSERACT_PATH", os.TempDir()+"\\definitely-not-a-file-ocrpdf.txt")
	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for missing Tesseract path")
	}
}

func TestHealthHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Errorf("body = %q, want status ok", rr.Body.String())
	}
}
