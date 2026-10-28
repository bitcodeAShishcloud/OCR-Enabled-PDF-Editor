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

	"ocrpdf/internal/api"
	"ocrpdf/internal/ocr"
)

// Config holds the parsed runtime configuration.
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

	if v := os.Getenv("TESSERACT_PATH"); v != "" {
		cfg.TesseractPath = v
	}
	if v := os.Getenv("FONT_PATH"); v != "" {
		cfg.FontPath = v
	}
	if v := os.Getenv("OCR_BACKEND"); v != "" {
		cfg.OCRBackend = v
	}

	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080"
	}
	p, err := strconv.Atoi(portStr)
	if err != nil || p < 1 || p > 65535 {
		return Config{}, fmt.Errorf("invalid PORT %q: must be an integer in 1..65535", portStr)
	}
	cfg.Port = p

	switch cfg.OCRBackend {
	case "cli", "gosseract":
	default:
		return Config{}, fmt.Errorf("invalid OCR_BACKEND %q (want cli or gosseract)", cfg.OCRBackend)
	}

	if _, err := os.Stat(cfg.TesseractPath); err != nil {
		return Config{}, fmt.Errorf("TESSERACT_PATH %q not found: %w", cfg.TesseractPath, err)
	}
	if _, err := os.Stat(cfg.FontPath); err != nil {
		return Config{}, fmt.Errorf("FONT_PATH %q not found: %w", cfg.FontPath, err)
	}

	return cfg, nil
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	engine, err := ocr.NewEngine(ocr.Config{
		Backend:       cfg.OCRBackend,
		TesseractPath: cfg.TesseractPath,
		FontPath:      cfg.FontPath,
	})
	if err != nil {
		slog.Error("ocr backend init failed", "err", err)
		os.Exit(1)
	}
	mux := newMux()

	distDir := os.Getenv("WEB_DIST")
	if distDir == "" {
		distDir = "web/dist"
	}

	store := api.NewStore()
	h := &api.Handler{
		Store:          store,
		Engine:         engine,
		FontPath:       cfg.FontPath,
		MaxUploadBytes: api.DefaultMaxUploadBytes,
		MaxPages:       api.DefaultMaxPages,
	}
	mux.HandleFunc("POST /api/documents", h.Upload)
	mux.HandleFunc("GET /api/documents/{id}/pages/{n}", h.GetPage)
	mux.HandleFunc("DELETE /api/documents/{id}", h.DeleteDoc)
	mux.HandleFunc("POST /api/documents/{id}/edits", h.Edit)
	mux.HandleFunc("GET /api/documents/{id}/result.pdf", h.Result)

	if info, err := os.Stat(distDir); err == nil && info.IsDir() {
		mux.Handle("GET /", spaHandler(distDir))
		slog.Info("serving spa", "dir", distDir)
	} else {
		slog.Info("web/dist not found; skipping spa serving", "dir", distDir)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

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
