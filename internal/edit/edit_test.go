package edit

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

func TestFitFontSize(t *testing.T) {
	fontPath := `C:\Windows\Fonts\arial.ttf`
	if _, err := os.Stat(fontPath); err != nil {
		t.Skipf("font unavailable: %v", err)
	}
	sz, err := FitFontSize(fontPath, "Jane", 100, 40)
	if err != nil {
		t.Fatalf("FitFontSize error: %v", err)
	}
	if sz < 4 || sz > 40 {
		t.Fatalf("fitted size %v out of range", sz)
	}
	face, err := gg.LoadFontFace(fontPath, sz)
	if err != nil {
		t.Fatal(err)
	}
	w := float64(font.MeasureString(face, "Jane") >> 6)
	h := float64(face.Metrics().Height) / 64.0
	if w > 100 || h > 40 {
		t.Fatalf("does not fit: w=%v h=%v", w, h)
	}
	got, err := FitFontSize(fontPath, "", 100, 40)
	if err != nil {
		t.Fatal(err)
	}
	if got != 4 {
		t.Fatalf("empty text size = %v, want 4", got)
	}
}

func TestEditTracer(t *testing.T) {
	fontPath := `C:\Windows\Fonts\arial.ttf`
	if _, err := os.Stat(fontPath); err != nil {
		t.Skipf("font unavailable: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 200, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.White)
		}
	}
	dc := gg.NewContextForRGBA(img)
	dc.SetColor(color.Black)
	if err := dc.LoadFontFace(fontPath, 32); err != nil {
		t.Fatal(err)
	}
	dc.DrawString("John", 10, 45)
	bbox := image.Rect(10, 13, 90, 47)
	bg := SampleBackground(img, bbox, 6)
	if bg.R < 240 || bg.G < 240 || bg.B < 240 {
		t.Fatalf("background not white-ish: %+v", bg)
	}
	tc := SampleTextColor(img, bbox, bg)
	if tc.R > 80 || tc.G > 80 || tc.B > 80 {
		t.Fatalf("text color not dark: %+v", tc)
	}
	Erase(img, bbox, 4, bg)
	sz, err := FitFontSize(fontPath, "Jane", float64(bbox.Dx()), float64(bbox.Dy()))
	if err != nil {
		t.Fatal(err)
	}
	if err := RenderText(img, bbox, "Jane", fontPath, sz, tc); err != nil {
		t.Fatal(err)
	}
	// image should still have dark pixels where Jane was drawn
	dark := 0
	for y := bbox.Min.Y; y < bbox.Max.Y; y++ {
		for x := bbox.Min.X; x < bbox.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if int(c.R)+int(c.G)+int(c.B) < 200 {
				dark++
			}
		}
	}
	if dark == 0 {
		t.Fatal("no dark pixels after render")
	}
}

