package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"ocrpdf/internal/ocr"
)

func makePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: 100, B: 50, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func makeScannedPDF(t *testing.T, dir string, pages int) string {
	t.Helper()
	imgs := make([]string, 0, pages)
	for i := 0; i < pages; i++ {
		p := filepath.Join(dir, fmt.Sprintf("p%d.png", i))
		makePNG(t, p, 400, 300)
		imgs = append(imgs, p)
	}
	pdfPath := filepath.Join(dir, "in.pdf")
	imp := pdfapi.DefaultImportConfig()
	imp.PageDim = &types.Dim{Width: 400, Height: 300}
	if err := pdfapi.ImportImagesFile(context.Background(), imgs, pdfPath, imp, model.NewStatelessConfiguration()); err != nil {
		t.Fatalf("import: %v", err)
	}
	return pdfPath
}

type fakeEngine struct {
	words []ocr.Word
	err   error
}

func (f fakeEngine) Recognize(string) ([]ocr.Word, error) {
	return f.words, f.err
}

func uploadFile(t *testing.T, h *Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/documents", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	h.Upload(rr, req)
	return rr
}

func TestUploadTracer(t *testing.T) {
	dir := t.TempDir()
	pdfPath := makeScannedPDF(t, dir, 1)

	engine, err := ocr.NewEngine(ocr.Config{Backend: "cli", TesseractPath: `C:\Program Files\Tesseract-OCR\tesseract.exe`})
	if err != nil {
		t.Skipf("ocr engine unavailable: %v", err)
	}
	h := &Handler{Store: NewStore(), Engine: engine, MaxUploadBytes: DefaultMaxUploadBytes, MaxPages: DefaultMaxPages}

	rr := uploadFile(t, h, pdfPath)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		DocID string `json:"docId"`
		Pages []struct {
			Page  int `json:"page"`
			Words []struct {
				Text string `json:"text"`
				Conf float64 `json:"conf"`
				BBox struct {
					X0, Y0, X1, Y1 int
				} `json:"bbox"`
			} `json:"words"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.DocID == "" {
		t.Fatal("empty docId")
	}
	if len(resp.Pages) != 1 {
		t.Fatalf("pages = %d, want 1", len(resp.Pages))
	}
}

func TestPreviewReturnsPNG(t *testing.T) {
	dir := t.TempDir()
	pdfPath := makeScannedPDF(t, dir, 1)
	engine, err := ocr.NewEngine(ocr.Config{Backend: "cli", TesseractPath: `C:\Program Files\Tesseract-OCR\tesseract.exe`})
	if err != nil {
		t.Skipf("ocr engine unavailable: %v", err)
	}
	h := &Handler{Store: NewStore(), Engine: engine, MaxUploadBytes: DefaultMaxUploadBytes, MaxPages: DefaultMaxPages}
	rr := uploadFile(t, h, pdfPath)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Body.String())
	}
	var resp struct {
		DocID string `json:"docId"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+resp.DocID+"/pages/1", nil)
	rr2 := httptest.NewRecorder()
	h.GetPage(rr2, req)
	// PathValue won't be set automatically; set via ServeMux? Use r with PathValue not available in direct call.
	_ = rr2
	// Instead use ServeMux routing to populate PathValue.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/documents/{id}/pages/{n}", h.GetPage)
	req2 := httptest.NewRequest(http.MethodGet, "/api/documents/"+resp.DocID+"/pages/1", nil)
	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, req2)
	if rr3.Code != http.StatusOK {
		t.Fatalf("preview status = %d, want 200; body=%s", rr3.Code, rr3.Body.String())
	}
	if ct := rr3.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	if rr3.Body.Len() == 0 {
		t.Fatal("empty body")
	}
}

func TestDeleteRemovesDocument(t *testing.T) {
	dir := t.TempDir()
	pdfPath := makeScannedPDF(t, dir, 1)
	engine, err := ocr.NewEngine(ocr.Config{Backend: "cli", TesseractPath: `C:\Program Files\Tesseract-OCR\tesseract.exe`})
	if err != nil {
		t.Skipf("ocr engine unavailable: %v", err)
	}
	h := &Handler{Store: NewStore(), Engine: engine, MaxUploadBytes: DefaultMaxUploadBytes, MaxPages: DefaultMaxPages}
	rr := uploadFile(t, h, pdfPath)
	var resp struct {
		DocID string `json:"docId"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/documents/{id}", h.DeleteDoc)
	mux.HandleFunc("GET /api/documents/{id}/pages/{n}", h.GetPage)
	mux.HandleFunc("GET /api/documents/{id}", func(w http.ResponseWriter, r *http.Request) { h.GetPage(w, r) })

	req := httptest.NewRequest(http.MethodDelete, "/api/documents/"+resp.DocID, nil)
	rrd := httptest.NewRecorder()
	mux.ServeHTTP(rrd, req)
	if rrd.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rrd.Code)
	}

	// subsequent GET page -> 404
	req2 := httptest.NewRequest(http.MethodGet, "/api/documents/"+resp.DocID+"/pages/1", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("after delete status = %d, want 404", rr2.Code)
	}
}

func TestUploadRejectsNonPDF(t *testing.T) {
	h := &Handler{Store: NewStore(), Engine: fakeEngine{}, MaxUploadBytes: DefaultMaxUploadBytes, MaxPages: DefaultMaxPages}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "x.txt")
	fw.Write([]byte("hello world"))
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/documents", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	h.Upload(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestUploadRejectsOversized(t *testing.T) {
	h := &Handler{Store: NewStore(), Engine: fakeEngine{}, MaxUploadBytes: 100, MaxPages: DefaultMaxPages}
	// build body larger than cap with pdf magic
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "big.pdf")
	fw.Write([]byte("%PDF-"))
	fw.Write(bytes.Repeat([]byte("a"), 200))
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/documents", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	h.Upload(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", rr.Code, rr.Body.String())
	}
}

func TestUploadRejectsTooManyPages(t *testing.T) {
	dir := t.TempDir()
	pdfPath := makeScannedPDF(t, dir, 2)
	h := &Handler{Store: NewStore(), Engine: fakeEngine{}, MaxUploadBytes: DefaultMaxUploadBytes, MaxPages: 1}
	rr := uploadFile(t, h, pdfPath)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}

func TestEmptyWordsSerializeAsArray(t *testing.T) {
	dir := t.TempDir()
	pdfPath := makeScannedPDF(t, dir, 1)
	h := &Handler{Store: NewStore(), Engine: fakeEngine{words: nil}, MaxUploadBytes: DefaultMaxUploadBytes, MaxPages: DefaultMaxPages}
	rr := uploadFile(t, h, pdfPath)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"words":[]`) {
		t.Fatalf("body does not contain words:[]: %s", rr.Body.String())
	}
}

func TestStoreConcurrent(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		id := fmt.Sprintf("id-%d", i)
		go func() {
			defer wg.Done()
			s.Add(&Document{ID: id, Dir: "/tmp", Pages: []Page{}})
		}()
		go func() {
			defer wg.Done()
			s.Get(id)
		}()
		go func() {
			defer wg.Done()
			s.Delete(id)
		}()
	}
	wg.Wait()
}

func TestUUIDFormat(t *testing.T) {
	id, err := newUUID()
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !re.MatchString(id) {
		t.Fatalf("uuid %q does not match v4 format", id)
	}
}