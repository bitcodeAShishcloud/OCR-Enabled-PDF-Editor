//go:build gosseract

package ocr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gosseract "github.com/otiai10/gosseract/v2"
)

// gosseractEngine is the optional CGO-backed OCR backend.
type gosseractEngine struct {
	tesseractPath  string
	tessdataPrefix string
}

func newGosseractEngine(cfg Config) (Engine, error) {
	prefix := os.Getenv("TESSDATA_PREFIX")
	if prefix == "" && cfg.TesseractPath != "" {
		prefix = filepath.Join(filepath.Dir(cfg.TesseractPath), "tessdata")
	}
	return &gosseractEngine{tesseractPath: cfg.TesseractPath, tessdataPrefix: prefix}, nil
}

// Recognize uses gosseract's GetBoundingBoxesVerbose to OCR the image.
func (g *gosseractEngine) Recognize(imagePath string) ([]Word, error) {
	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetImage(imagePath); err != nil {
		return nil, fmt.Errorf("ocr: gosseract set image: %w", err)
	}
	if g.tessdataPrefix != "" {
		if err := client.SetTessdataPrefix(g.tessdataPrefix); err != nil {
			return nil, fmt.Errorf("ocr: gosseract set tessdata prefix: %w", err)
		}
	}
	boxes, err := client.GetBoundingBoxesVerbose()
	if err != nil {
		return nil, fmt.Errorf("ocr: gosseract recognize: %w", err)
	}

	words := make([]Word, 0, len(boxes))
	for _, b := range boxes {
		text := strings.TrimSpace(b.Word)
		if text == "" {
			continue
		}
		if b.Confidence < 0 {
			continue
		}
		words = append(words, Word{
			Text:        b.Word,
			Confidence:  b.Confidence,
			X0:          b.Box.Min.X,
			Y0:          b.Box.Min.Y,
			X1:          b.Box.Max.X,
			Y1:          b.Box.Max.Y,
		})
	}
	return words, nil
}