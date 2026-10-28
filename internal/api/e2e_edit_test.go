package api

import (
	"bytes"
	"context"
	"encoding/json"
	"image/color"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/fogleman/gg"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"ocrpdf/internal/ocr"
	"ocrpdf/internal/pdfio"
)

func TestEditRebuildRoundTrip(t *testing.T) {
	fontPath := `C:\Windows\Fonts\arial.ttf`
	tessPath := `C:\Program Files\Tesseract-OCR\tesseract.exe`
	if _, err := os.Stat(fontPath); err != nil {
		t.Skipf("font unavailable: %v", err)
	}
	if _, err := os.Stat(tessPath); err != nil {
		t.Skipf("tesseract unavailable: %v", err)
	}

	dir := t.TempDir()

	// 1. render name onto image
	imgPath := filepath.Join(dir, "src.png")
	dc := gg.NewContext(400, 120)
	dc.SetColor(color.White)
	dc.Clear()
	dc.SetColor(color.Black)
	if err := dc.LoadFontFace(fontPath, 48); err != nil {
		t.Fatal(err)
	}
	dc.DrawString("John", 20, 80)
	if err := dc.SavePNG(imgPath); err != nil {
		t.Fatal(err)
	}

	// 2. import to PDF
	pdfPath := filepath.Join(dir, "in.pdf")
	imp := pdfapi.DefaultImportConfig()
	imp.PageDim = &types.Dim{Width: 400, Height: 120}
	if err := pdfapi.ImportImagesFile(context.Background(), []string{imgPath}, pdfPath, imp, model.NewStatelessConfiguration()); err != nil {
		t.Fatalf("import: %v", err)
	}

	// 3. upload + OCR
	engine, err := ocr.NewEngine(ocr.Config{Backend: "cli", TesseractPath: tessPath})
	if err != nil {
		t.Skipf("ocr engine unavailable: %v", err)
	}
	h := &Handler{Store: NewStore(), Engine: engine, FontPath: fontPath, MaxUploadBytes: DefaultMaxUploadBytes, MaxPages: DefaultMaxPages}

	rr := uploadFile(t, h, pdfPath)
	if rr.Code != http.StatusOK {
		t.Fatalf("upload status = %d; body=%s", rr.Code, rr.Body.String())
	}
	var up struct {
		DocID string `json:"docId"`
		Pages []struct {
			Words []struct {
				Text string `json:"text"`
			} `json:"words"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &up); err != nil {
		t.Fatal(err)
	}
	if up.DocID == "" {
		t.Fatal("empty docId")
	}
	// verify OCR saw John
	foundJohn := false
	for _, p := range up.Pages {
		for _, w := range p.Words {
			if strings.EqualFold(w.Text, "John") {
				foundJohn = true
			}
		}
	}
	if !foundJohn {
		t.Fatalf("initial OCR did not detect John; body=%s", rr.Body.String())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/documents/{id}/edits", h.Edit)
	mux.HandleFunc("GET /api/documents/{id}/result.pdf", h.Result)

	// 4. POST edit
	body := bytes.NewBufferString(`{"page":1,"oldText":"John","newText":"Jane"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+up.DocID+"/edits", body)
	req.Header.Set("Content-Type", "application/json")
	rre := httptest.NewRecorder()
	mux.ServeHTTP(rre, req)
	if rre.Code != http.StatusOK {
		t.Fatalf("edit status = %d; body=%s", rre.Code, rre.Body.String())
	}

	// 5. GET result.pdf
	req2 := httptest.NewRequest(http.MethodGet, "/api/documents/"+up.DocID+"/result.pdf", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("result status = %d; body=%s", rr2.Code, rr2.Body.String())
	}
	if ct := rr2.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", ct)
	}
	resultPath := filepath.Join(dir, "result.pdf")
	if err := os.WriteFile(resultPath, rr2.Body.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	// 6. extract + OCR
	imgs, err := pdfio.ExtractImages(context.Background(), resultPath, filepath.Join(dir, "out"))
	if err != nil {
		t.Fatalf("extract result: %v", err)
	}
	if len(imgs) == 0 {
		t.Fatal("no images extracted from result.pdf")
	}
	words, err := engine.Recognize(imgs[0])
	if err != nil {
		t.Fatalf("re-OCR: %v", err)
	}
	texts := make([]string, 0, len(words))
	for _, w := range words {
		texts = append(texts, w.Text)
	}
	joined := strings.Join(texts, " ")
	t.Logf("re-OCR: %s", joined)
	if !strings.Contains(strings.ToLower(joined), "jane") {
		t.Fatalf("re-OCR did not contain Jane; got %q", joined)
	}
}
