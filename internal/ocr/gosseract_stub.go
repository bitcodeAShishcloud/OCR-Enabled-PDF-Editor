//go:build !gosseract

package ocr

import "errors"

func newGosseractEngine(cfg Config) (Engine, error) {
	return nil, errors.New("gosseract backend requires building with -tags gosseract")
}
