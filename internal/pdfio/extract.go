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
	"regexp"
	"strconv"

	_ "github.com/hhrutter/tiff"
	_ "golang.org/x/image/webp"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

var pageFileRe = regexp.MustCompile(`_(\d+)_([A-Za-z0-9]+)\.[A-Za-z0-9]+$`)

func ExtractImages(ctx context.Context, pdfPath, outDir string) ([]string, error) {
	if pdfPath == "" {
		return nil, fmt.Errorf("pdfio: pdfPath must not be empty")
	}
	if outDir == "" {
		return nil, fmt.Errorf("pdfio: outDir must not be empty")
	}
	pdfPath = filepath.Clean(pdfPath)
	outDir = filepath.Clean(outDir)

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("pdfio: create output dir %q: %w", outDir, err)
	}
	// Clear stale artifacts from a previous run.
	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(outDir, e.Name()))
	}

	if err := pdfapi.ExtractImagesFile(ctx, pdfPath, outDir, nil, model.NewStatelessConfiguration()); err != nil {
		return nil, fmt.Errorf("pdfio: extract images from %q: %w", pdfPath, err)
	}

	pageCount, err := pdfapi.PageCountFile(ctx, pdfPath)
	if err != nil {
		return nil, fmt.Errorf("pdfio: page count for %q: %w", pdfPath, err)
	}

	byPage := make(map[int][]string)
	files, err := os.ReadDir(outDir)
	if err != nil {
		return nil, fmt.Errorf("pdfio: read output dir %q: %w", outDir, err)
	}
	for _, e := range files {
		if e.IsDir() {
			continue
		}
		m := pageFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		page, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		byPage[page] = append(byPage[page], filepath.Join(outDir, e.Name()))
	}

	out := make([]string, 0, pageCount)
	for page := 1; page <= pageCount; page++ {
		candidates := byPage[page]
		if len(candidates) == 0 {
			return nil, fmt.Errorf("pdfio: no embedded image for page %d", page)
		}
		best, err := largestImage(candidates)
		if err != nil {
			return nil, fmt.Errorf("pdfio: select image for page %d: %w", page, err)
		}
		out = append(out, best)
	}
	return out, nil
}

func largestImage(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("pdfio: no candidate images")
	}
	type candidate struct {
		path string
		area int
		size int64
	}
	var best *candidate
	for _, p := range paths {
		area := 0
		f, err := os.Open(p)
		if err == nil {
			cfg, _, derr := image.DecodeConfig(f)
			f.Close()
			if derr == nil {
				area = cfg.Width * cfg.Height
			}
		}
		var size int64
		if info, err := os.Stat(p); err == nil {
			size = info.Size()
		}
		c := &candidate{path: p, area: area, size: size}
		if best == nil ||
			c.area > best.area ||
			(c.area == best.area && c.size > best.size) ||
			(c.area == best.area && c.size == best.size && c.path < best.path) {
			best = c
		}
	}
	if best == nil {
		return "", fmt.Errorf("pdfio: no decodable image")
	}
	return best.path, nil
}
