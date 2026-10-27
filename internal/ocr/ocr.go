package ocr

import "fmt"

// Word represents a single recognized word with its bounding box.
type Word struct {
	Text        string
	Confidence  float64
	X0, Y0, X1, Y1 int
}

// Engine is the OCR abstraction used by the API and edit layers.
type Engine interface {
	Recognize(imagePath string) ([]Word, error)
}

// Config carries the OCR backend selection and related paths.
type Config struct {
	Backend       string
	TesseractPath string
	FontPath      string
}

// NewEngine returns the OCR engine selected by cfg.Backend.
func NewEngine(cfg Config) (Engine, error) {
	switch cfg.Backend {
	case "cli":
		return &cliEngine{tesseractPath: cfg.TesseractPath}, nil
	case "gosseract":
		return newGosseractEngine(cfg)
	default:
		return nil, fmt.Errorf("invalid OCR backend %q (want cli or gosseract)", cfg.Backend)
	}
}
