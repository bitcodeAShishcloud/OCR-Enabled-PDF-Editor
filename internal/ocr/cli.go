package ocr

import "errors"

// cliEngine is the default OCR backend that shells out to the tesseract binary.
type cliEngine struct {
	tesseractPath string
}

// Recognize is implemented in Phase 3.
func (c *cliEngine) Recognize(imagePath string) ([]Word, error) {
	return nil, errors.New("cli OCR backend not implemented (Phase 3)")
}
