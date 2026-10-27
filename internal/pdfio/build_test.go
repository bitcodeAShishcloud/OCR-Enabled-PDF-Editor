package pdfio

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestBuildPageCount(t *testing.T) {
	dir := t.TempDir()
	paths := make([]string, 3)
	for i := range paths {
		p := filepath.Join(dir, "img"+string(rune('a'+i))+".png")
		writePNG(t, p, 40, 30)
		paths[i] = p
	}
	out := filepath.Join(dir, "out.pdf")
	if err := BuildPDF(context.Background(), paths, out); err != nil {
		t.Fatal(err)
	}
	n, err := pdfapi.PageCountFile(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("page count %d want 3", n)
	}
}

func TestBuildNoAppend(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.png")
	p2 := filepath.Join(dir, "b.png")
	writePNG(t, p1, 40, 30)
	writePNG(t, p2, 40, 30)
	out := filepath.Join(dir, "out.pdf")
	if err := BuildPDF(context.Background(), []string{p1, p2}, out); err != nil {
		t.Fatal(err)
	}
	// rebuild with 1 image
	if err := BuildPDF(context.Background(), []string{p1}, out); err != nil {
		t.Fatal(err)
	}
	n, err := pdfapi.PageCountFile(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("page count %d want 1", n)
	}
}

func TestBuildPageDim(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.png")
	p2 := filepath.Join(dir, "b.png")
	writePNG(t, p1, 100, 60)
	writePNG(t, p2, 100, 60)
	out := filepath.Join(dir, "out.pdf")
	if err := BuildPDF(context.Background(), []string{p1, p2}, out); err != nil {
		t.Fatal(err)
	}
	dims, err := pdfapi.PageDimsFile(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	if len(dims) != 2 {
		t.Fatalf("len dims %d", len(dims))
	}
	for i, d := range dims {
		if d.Width != 100 || d.Height != 60 {
			t.Fatalf("page %d dim %.0fx%.0f want 100x60", i, d.Width, d.Height)
		}
	}
	_ = types.Dim{}
	_ = model.NewStatelessConfiguration
}

func TestBuildRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.pdf")
	if err := BuildPDF(context.Background(), nil, out); err == nil {
		t.Fatal("expected error for empty imagePaths")
	}
	if err := BuildPDF(context.Background(), []string{}, out); err == nil {
		t.Fatal("expected error for empty imagePaths")
	}
	if err := BuildPDF(context.Background(), []string{"x.png"}, ""); err == nil {
		t.Fatal("expected error for empty outputPath")
	}
}

func TestNewTempDirCreatesAndCleans(t *testing.T) {
	dir, cleanup, err := NewTempDir("test-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("dir not exists: %v", err)
	}
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("dir still exists")
	}
}