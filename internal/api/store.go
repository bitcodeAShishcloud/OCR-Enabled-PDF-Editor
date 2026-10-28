package api

import (
	"sync"

	"ocrpdf/internal/ocr"
)

// Page holds a single extracted page image and its OCR words.
type Page struct {
	Number    int
	ImagePath string
	Words     []ocr.Word
}

// Document is mutable after insertion: Page.ImagePath is overwritten on edit.
// mu guards Pages (and any per-document state). Zero value is usable.
// Lock ordering: store mutex -> doc mutex, never reverse. Never hold doc mutex
// across Tesseract/pdfcpu calls except for the full edit mutation.
type Document struct {
	mu    sync.Mutex
	ID    string
	Dir   string
	Pages []Page
}

// Store is a mutex-protected in-memory document map.
type Store struct {
	mu   sync.Mutex
	docs map[string]*Document
}

func NewStore() *Store {
	return &Store{docs: make(map[string]*Document)}
}

func (s *Store) Add(doc *Document) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[doc.ID] = doc
}

func (s *Store) Get(id string) (*Document, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.docs[id]
	return d, ok
}

func (s *Store) Delete(id string) (*Document, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.docs[id]
	if ok {
		delete(s.docs, id)
	}
	return d, ok
}