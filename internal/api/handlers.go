package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"

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
	if n > len(doc.Pages) {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	imgPath := doc.Pages[n-1].ImagePath

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

