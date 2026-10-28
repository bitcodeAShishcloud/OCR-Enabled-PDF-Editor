package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupDist(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!DOCTYPE html><html><body>INDEX</body></html>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("console.log('hi');"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSPAHandlerServesFile(t *testing.T) {
	dir := setupDist(t)
	h := spaHandler(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "INDEX") {
		t.Errorf("GET / body = %q, want index", rr.Body.String())
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("GET /app.js status = %d, want 200", rr2.Code)
	}
	if rr2.Body.String() != "console.log('hi');" {
		t.Errorf("GET /app.js body = %q", rr2.Body.String())
	}
}

func TestSPAHandlerFallback(t *testing.T) {
	dir := setupDist(t)
	h := spaHandler(dir)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/documents/abc", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "INDEX") {
		t.Errorf("body = %q, want index fallback", rr.Body.String())
	}
}

func TestSPAHandlerAPIPassthrough(t *testing.T) {
	dir := setupDist(t)
	h := spaHandler(dir)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/foo", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestSPAHandlerMissingDir(t *testing.T) {
	h := spaHandler(filepath.Join(os.TempDir(), "does-not-exist-ocrpdf"))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

