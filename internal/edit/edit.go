package edit

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
)

// ToRGBA returns img as *image.RGBA, copying if necessary.
func ToRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok {
		return r
	}
	b := img.Bounds()
	r := image.NewRGBA(b)
	draw.Draw(r, b, img, b.Min, draw.Src)
	return r
}

// ParseColor parses a #RGB or #RRGGBB color string into an opaque color.RGBA.
func ParseColor(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return color.RGBA{}, fmt.Errorf("edit: empty color")
	}
	if !strings.HasPrefix(s, "#") {
		return color.RGBA{}, fmt.Errorf("edit: color must start with #")
	}
	s = s[1:]
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("edit: invalid color length %d", len(s))
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("edit: invalid hex color: %w", err)
	}
	return color.RGBA{R: uint8((v >> 16) & 0xff), G: uint8((v >> 8) & 0xff), B: uint8(v & 0xff), A: 255}, nil
}
