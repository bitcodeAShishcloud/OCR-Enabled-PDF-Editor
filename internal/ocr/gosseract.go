//go:build gosseract

package ocr

import "errors"

// gosseractEngine is the optional CGO-backed OCR backend.
type gosseractEngine struct {
	tesseractPath string
}

func newGosseractEngine(cfg Config) (Engine, error) {
	return &gosseractEngine{tesseractPath: cfg.TesseractPath}, nil
}

// Recognize is implemented in Phase 3.
func (g *gosseractEngine) Recognize(imagePath string) ([]Word, error) {
	return nil, errors.New("gosseract OCR backend not implemented (Phase 3)")
}
