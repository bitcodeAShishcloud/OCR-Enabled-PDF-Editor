package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"

	"ocrpdf/internal/edit"
	"ocrpdf/internal/ocr"
	"ocrpdf/internal/pdfio"
)

const (
	DefaultMaxUploadBytes = 50 * 1024 * 1024
	DefaultMaxPages       = 50
)

type Handler struct {
	Store           *Store
	Engine          ocr.Engine
	FontPath        string
	MaxUploadBytes  int64
	MaxPages        int
}

type bboxDTO struct {
	X0 int `json:"x0"`
	Y0 int `json:"y0"`
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
}

type wordDTO struct {
	Text string  `json:"text"`
	Conf float64 `json:"conf"`
	BBox bboxDTO `json:"bbox"`
}

type pageDTO struct {
	Page  int      `json:"page"`
	Words []wordDTO `json:"words"`
}

type uploadResponse struct {
	DocID string   `json:"docId"`
	Pages []pageDTO `json:"pages"`
}

func toWordDTOs(words []ocr.Word) []wordDTO {
	out := make([]wordDTO, 0, len(words))
	for _, w := range words {
		out = append(out, wordDTO{
			Text: w.Text,
			Conf: w.Confidence,
			BBox: bboxDTO{X0: w.X0, Y0: w.Y0, X1: w.X1, Y1: w.Y1},
		})
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Upload is the exported handler for POST /api/documents.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) { h.upload(w, r) }

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.MaxUploadBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if isMaxBytesErr(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	f, _, err := r.FormFile("file")
	if err != nil {
		// Fallback to first file part.
		if r.MultipartForm != nil && len(r.MultipartForm.File) > 0 {
			for _, files := range r.MultipartForm.File {
				if len(files) > 0 {
					fh := files[0]
					opened, oerr := fh.Open()
					if oerr != nil {
						writeError(w, http.StatusInternalServerError, "open file part failed")
						return
					}
					f = opened
					_ = fh
					break
				}
			}
		}
		if f == nil {
			writeError(w, http.StatusBadRequest, "missing 'file' part")
			return
		}
	}
	defer f.Close()

	var magic [5]byte
	n, _ := f.Read(magic[:])
	if n < 5 || string(magic[:]) != "%PDF-" {
		writeError(w, http.StatusBadRequest, "uploaded file is not a PDF")
		return
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, "seek failed")
		return
	}

	dir, err := os.MkdirTemp("", "ocrpdf-")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "temp dir creation failed")
		return
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(dir)
		}
	}()

	pdfPath := filepath.Join(dir, "input.pdf")
	dst, err := os.Create(pdfPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "file creation failed")
		return
	}
	if _, err := io.Copy(dst, f); err != nil {
		dst.Close()
		if isMaxBytesErr(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "upload exceeds size cap")
			return
		}
		writeError(w, http.StatusInternalServerError, "file write failed")
		return
	}
	dst.Close()

	ctx := r.Context()
	pageCount, err := pdfapi.PageCountFile(ctx, pdfPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or unreadable PDF")
		return
	}
	if pageCount == 0 {
		writeError(w, http.StatusBadRequest, "PDF has no pages")
		return
	}
	if pageCount > h.MaxPages {
		writeError(w, http.StatusBadRequest, "page count exceeds cap")
		return
	}

	imgDir := filepath.Join(dir, "pages")
	imagePaths, err := pdfio.ExtractImages(ctx, pdfPath, imgDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "image extraction failed")
		return
	}

	id, err := newUUID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "id generation failed")
		return
	}

	pages := make([]Page, 0, len(imagePaths))
	pageDTOs := make([]pageDTO, 0, len(imagePaths))
	for i, p := range imagePaths {
		words, err := h.Engine.Recognize(p)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "OCR failed")
			return
		}
		pages = append(pages, Page{Number: i + 1, ImagePath: p, Words: words})
		pageDTOs = append(pageDTOs, pageDTO{
			Page:  i + 1,
			Words: toWordDTOs(words),
		})
	}

	h.Store.Add(&Document{ID: id, Dir: dir, Pages: pages})
	success = true

	writeJSON(w, http.StatusOK, uploadResponse{DocID: id, Pages: pageDTOs})
}

func isMaxBytesErr(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

// GetPage is the exported handler for GET /api/documents/{id}/pages/{n}.
func (h *Handler) GetPage(w http.ResponseWriter, r *http.Request) { h.getPage(w, r) }

func (h *Handler) getPage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 1 {
		writeError(w, http.StatusBadRequest, "invalid page number")
		return
	}

	doc, ok := h.Store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	doc.mu.Lock()
	if n > len(doc.Pages) {
		doc.mu.Unlock()
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	imgPath := doc.Pages[n-1].ImagePath
	doc.mu.Unlock()

	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, imgPath)
}

// DeleteDoc is the exported handler for DELETE /api/documents/{id}.
func (h *Handler) DeleteDoc(w http.ResponseWriter, r *http.Request) { h.deleteDoc(w, r) }

func (h *Handler) deleteDoc(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	doc, ok := h.Store.Delete(id)
	if !ok {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err := os.RemoveAll(doc.Dir); err != nil {
		slog.Error("cleanup failed", "id", id, "err", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

type editRequest struct {
	Page     int       `json:"page"`
	OldText  string    `json:"oldText"`
	WordIDs  []int     `json:"wordIds"`
	NewText  string    `json:"newText"`
	FontPath string    `json:"fontPath"`
	FontSize *float64  `json:"fontSize"`
	Color    string    `json:"color"`
}

// Edit is the exported handler for POST /api/documents/{id}/edits.
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) { h.edit(w, r) }

func (h *Handler) edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req editRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Page < 1 {
		writeError(w, http.StatusBadRequest, "page required")
		return
	}
	if req.NewText == "" {
		writeError(w, http.StatusBadRequest, "newText required")
		return
	}
	if req.OldText == "" && len(req.WordIDs) == 0 {
		writeError(w, http.StatusBadRequest, "oldText or wordIds required")
		return
	}

	doc, ok := h.Store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	doc.mu.Lock()
	defer doc.mu.Unlock()

	if req.Page > len(doc.Pages) {
		writeError(w, http.StatusBadRequest, "page not found")
		return
	}
	page := &doc.Pages[req.Page-1]

	// Resolve target words.
	var targets []ocr.Word
	if len(req.WordIDs) > 0 {
		for _, idx := range req.WordIDs {
			if idx < 0 || idx >= len(page.Words) {
				writeError(w, http.StatusBadRequest, "wordId out of range")
				return
			}
			targets = append(targets, page.Words[idx])
		}
	} else {
		for _, wd := range page.Words {
			if wd.Text == req.OldText {
				targets = append(targets, wd)
				break
			}
		}
		if len(targets) == 0 {
			writeError(w, http.StatusBadRequest, "oldText not found")
			return
		}
	}

	// Union bbox.
	bbox := image.Rect(targets[0].X0, targets[0].Y0, targets[0].X1, targets[0].Y1)
	for _, wd := range targets[1:] {
		bbox = bbox.Union(image.Rect(wd.X0, wd.Y0, wd.X1, wd.Y1))
	}
	bbox = bbox.Intersect(image.Rect(0, 0, 1<<30, 1<<30))

	// Load current page image.
	f, err := os.Open(page.ImagePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "open image failed")
		return
	}
	img, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "decode image failed")
		return
	}
	rgba := edit.ToRGBA(img)

	// Sample colors before erase.
	bg := edit.SampleBackground(rgba, bbox, 6)
	tc := edit.SampleTextColor(rgba, bbox, bg)

	// Font path override (validated).
	fontPath := h.FontPath
	if req.FontPath != "" {
		if _, err := os.Stat(req.FontPath); err != nil {
			writeError(w, http.StatusBadRequest, "font path not found")
			return
		}
		fontPath = req.FontPath
	}
	// Color override.
	if req.Color != "" {
		c, err := edit.ParseColor(req.Color)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid color")
			return
		}
		tc = c
	}

	// Font size.
	size := 0.0
	if req.FontSize != nil {
		size = *req.FontSize
	}
	if size > 0 {
		// clamp to box height/width to avoid clipping
		face, err := gg.LoadFontFace(fontPath, size)
		if err == nil {
			hpx := float64(face.Metrics().Height) / 64.0
			if hpx > float64(bbox.Dy()) && hpx > 0 {
				size = size * float64(bbox.Dy()) / hpx
			}
			wpx := float64(font.MeasureString(face, req.NewText) >> 6)
			if wpx > float64(bbox.Dx()) && wpx > 0 {
				size = size * float64(bbox.Dx()) / wpx
			}
		}
	}

	// Erase + render.
	edit.Erase(rgba, bbox, 4, bg)
	if err := edit.RenderText(rgba, bbox, req.NewText, fontPath, size, tc); err != nil {
		writeError(w, http.StatusInternalServerError, "render failed")
		return
	}

	// Persist edited image.
	editedPath := filepath.Join(doc.Dir, "edited_p"+strconv.Itoa(req.Page)+".png")
	out, err := os.Create(editedPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create edited image failed")
		return
	}
	if err := png.Encode(out, rgba); err != nil {
		out.Close()
		writeError(w, http.StatusInternalServerError, "encode edited image failed")
		return
	}
	out.Close()
	page.ImagePath = editedPath

	// Return preview bytes.
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		writeError(w, http.StatusInternalServerError, "encode preview failed")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(buf.Bytes())
}

// Result is the exported handler for GET /api/documents/{id}/result.pdf.
func (h *Handler) Result(w http.ResponseWriter, r *http.Request) { h.result(w, r) }

func (h *Handler) result(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	doc, ok := h.Store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	doc.mu.Lock()
	paths := make([]string, len(doc.Pages))
	for i, p := range doc.Pages {
		paths[i] = p.ImagePath
	}
	doc.mu.Unlock()

	out := filepath.Join(doc.Dir, "result.pdf")
	if err := pdfio.BuildPDF(r.Context(), paths, out); err != nil {
		writeError(w, http.StatusInternalServerError, "rebuild failed")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	http.ServeFile(w, r, out)
}

