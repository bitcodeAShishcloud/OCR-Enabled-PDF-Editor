package edit

import (
	"image"
	"image/color"

	"github.com/fogleman/gg"
)

// Erase fills the bbox expanded by pad with bg, clamped to image bounds.
func Erase(img *image.RGBA, bbox image.Rectangle, pad int, bg color.RGBA) {
	if pad < 0 {
		pad = 0
	}
	r := bbox.Inset(-pad).Intersect(img.Bounds())
	if r.Empty() {
		return
	}
	dc := gg.NewContextForRGBA(img)
	dc.SetColor(bg)
	dc.DrawRectangle(float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()))
	dc.Fill()
}
