package pdfio

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	_ "github.com/hhrutter/tiff"
	_ "golang.org/x/image/webp"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func BuildPDF(ctx context.Context, imagePaths []string, outputPath string) error {
	if len(imagePaths) == 0 {
		return fmt.Errorf("pdfio: no images to build")
	}
	if outputPath == "" {
		return fmt.Errorf("pdfio: outputPath must not be empty")
	}
	outputPath = filepath.Clean(outputPath)

	if dir := filepath.Dir(outputPath); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("pdfio: create output dir %q: %w", dir, err)
		}
	}

	// PDF-03: truncate any existing output file (no append).
	if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("pdfio: remove existing output %q: %w", outputPath, err)
	}

	// PDF-04: page dimensions from the first source image.
	f, err := os.Open(imagePaths[0])
	if err != nil {
		return fmt.Errorf("pdfio: open %q: %w", imagePaths[0], err)
	}
	cfg, _, err := image.DecodeConfig(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("pdfio: decode config %q: %w", imagePaths[0], err)
	}

	imp := pdfapi.DefaultImportConfig() // D-06
	imp.PageDim = &types.Dim{Width: float64(cfg.Width), Height: float64(cfg.Height)} // D-05

	if err := pdfapi.ImportImagesFile(ctx, imagePaths, outputPath, imp, model.NewStatelessConfiguration()); err != nil {
		return fmt.Errorf("pdfio: build pdf %q: %w", outputPath, err)
	}
	return nil
}
