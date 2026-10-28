package edit

import (
	"image"
	"image/color"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

// FitFontSize returns the largest font size (in points) such that text fits both
// width and height of the box. Returns 4 when text is empty.
func FitFontSize(fontPath, text string, boxW, boxH float64) (float64, error) {
	if text == "" {
		return 4, nil
	}
	if boxW <= 0 || boxH <= 0 {
		return 4, nil
	}
	lo, hi := 4.0, boxH
	if hi > 200 {
		hi = 200
	}
	if hi < 4 {
		hi = 4
	}
	best := 4.0
	for i := 0; i < 40; i++ {
		mid := (lo + hi) / 2
		face, err := gg.LoadFontFace(fontPath, mid)
		if err != nil {
			return 0, err
		}
		w := float64(font.MeasureString(face, text) >> 6)
		h := float64(face.Metrics().Height) / 64.0
		if w <= boxW && h <= boxH {
			best = mid
			lo = mid
		} else {
			hi = mid
		}
	}
	return best, nil
}

// RenderText draws text into bbox using the given font size (0 means auto-fit),
// sampling baseline from font metrics so ascenders/descenders stay inside bbox.
func RenderText(img *image.RGBA, bbox image.Rectangle, text string, fontPath string, size float64, col color.Color) error {
	if size <= 0 {
		sz, err := FitFontSize(fontPath, text, float64(bbox.Dx()), float64(bbox.Dy()))
		if err != nil {
			return err
		}
		size = sz
	}
	face, err := gg.LoadFontFace(fontPath, size)
	if err != nil {
		return err
	}
	m := face.Metrics()
	baseline := float64(bbox.Max.Y) - float64(m.Descent)/64.0
	ascent := float64(m.Ascent) / 64.0
	if baseline-ascent < float64(bbox.Min.Y) {
		baseline = float64(bbox.Min.Y) + ascent
	}
	dc := gg.NewContextForRGBA(img)
	dc.SetColor(col)
	if err := dc.LoadFontFace(fontPath, size); err != nil {
		return err
	}
	dc.DrawString(text, float64(bbox.Min.X), baseline)
	_ = face
	return nil
}
