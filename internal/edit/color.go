package edit

import (
	"image"
	"image/color"
	"sort"
)

func luminance(c color.RGBA) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

func medianInts(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]int(nil), xs...)
	sort.Ints(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

// SampleBackground samples the background color from the pixel ring around bbox.
// Pixels darker than a luminance threshold (ink) are excluded; per-channel median
// is used for robustness.
func SampleBackground(img *image.RGBA, bbox image.Rectangle, ring int) color.RGBA {
	if ring <= 0 {
		ring = 6
	}
	bounds := img.Bounds()
	outer := bbox.Inset(-ring).Intersect(bounds)
	var rs, gs, bs []int
	for y := outer.Min.Y; y < outer.Max.Y; y++ {
		for x := outer.Min.X; x < outer.Max.X; x++ {
			if image.Pt(x, y).In(bbox) {
				continue
			}
			c := img.RGBAAt(x, y)
			if luminance(c) < 80 {
				continue
			}
			rs = append(rs, int(c.R))
			gs = append(gs, int(c.G))
			bs = append(bs, int(c.B))
		}
	}
	if len(rs) == 0 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return color.RGBA{
		R: uint8(medianInts(rs)),
		G: uint8(medianInts(gs)),
		B: uint8(medianInts(bs)),
		A: 255,
	}
}

// SampleTextColor samples the original glyph color from the darkest pixels inside
// bbox. Falls back to the darkest 10% of bbox pixels when too few dark pixels
// exist, and to opaque black when none.
func SampleTextColor(img *image.RGBA, bbox image.Rectangle, bg color.RGBA) color.RGBA {
	bgLum := luminance(bg)
	var rs, gs, bs []int
	for y := bbox.Min.Y; y < bbox.Max.Y; y++ {
		for x := bbox.Min.X; x < bbox.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if luminance(c) < bgLum-30 {
				rs = append(rs, int(c.R))
				gs = append(gs, int(c.G))
				bs = append(bs, int(c.B))
			}
		}
	}
	if len(rs) < 5 {
		// fallback: darkest 10% of all bbox pixels
		type px struct{ r, g, b int; lum float64 }
		var all []px
		for y := bbox.Min.Y; y < bbox.Max.Y; y++ {
			for x := bbox.Min.X; x < bbox.Max.X; x++ {
				c := img.RGBAAt(x, y)
				all = append(all, px{int(c.R), int(c.G), int(c.B), luminance(c)})
			}
		}
		if len(all) == 0 {
			return color.RGBA{R: 0, G: 0, B: 0, A: 255}
		}
		sort.Slice(all, func(i, j int) bool { return all[i].lum < all[j].lum })
		n := len(all) / 10
		if n < 1 {
			n = 1
		}
		rs, gs, bs = nil, nil, nil
		for i := 0; i < n; i++ {
			rs = append(rs, all[i].r)
			gs = append(gs, all[i].g)
			bs = append(bs, all[i].b)
		}
	}
	if len(rs) == 0 {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}
	return color.RGBA{
		R: uint8(medianInts(rs)),
		G: uint8(medianInts(gs)),
		B: uint8(medianInts(bs)),
		A: 255,
	}
}
