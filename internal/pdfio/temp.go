package pdfio

import (
	"fmt"
	"os"
)

func NewTempDir(prefix string) (string, func(), error) {
	if prefix == "" {
		prefix = "ocrpdf-"
	}
	dir, err := os.MkdirTemp("", prefix+"*")
	if err != nil {
		return "", nil, fmt.Errorf("pdfio: create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	return dir, cleanup, nil
}
