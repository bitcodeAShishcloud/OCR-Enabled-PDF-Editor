package ocr

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	xdraw "golang.org/x/image/draw"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultTesseractPath is the Windows default location of the tesseract binary.
const DefaultTesseractPath = `C:\Program Files\Tesseract-OCR\tesseract.exe`

// cliEngine is the default OCR backend that shells out to the tesseract binary.
type cliEngine struct {
	tesseractPath string
}

// ParseTSV parses Tesseract's TSV output into Word structs.
func ParseTSV(r io.Reader) ([]Word, error) {
	cr := csv.NewReader(r)
	cr.Comma = '\t'
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true

	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ocr: read tsv: %w", err)
	}

	words := make([]Word, 0)
	for _, rec := range records {
		if len(rec) == 0 {
			continue
		}
		if strings.TrimSpace(rec[0]) == "level" {
			continue
		}
		if rec[0] != "5" {
			continue
		}
		if len(rec) < 11 {
			continue
		}
		conf, err := strconv.ParseFloat(strings.TrimSpace(rec[10]), 64)
		if err != nil {
			continue
		}
		if conf < 0 {
			continue
		}
		left, err1 := strconv.Atoi(strings.TrimSpace(rec[6]))
		top, err2 := strconv.Atoi(strings.TrimSpace(rec[7]))
		w, err3 := strconv.Atoi(strings.TrimSpace(rec[8]))
		h, err4 := strconv.Atoi(strings.TrimSpace(rec[9]))
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}
		text := ""
		if len(rec) >= 12 {
			text = rec[11]
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		words = append(words, Word{
			Text:        text,
			Confidence:  conf,
			X0:          left,
			Y0:          top,
			X1:          left + w,
			Y1:          top + h,
		})
	}
	return words, nil
}

// Recognize runs tesseract on the given image and parses its TSV output.
// Small images are upscaled 2x before OCR to improve accuracy; bounding boxes
// are scaled back to the original image coordinates.
func (c *cliEngine) Recognize(imagePath string) ([]Word, error) {
	bin := c.tesseractPath
	if bin == "" {
		bin = DefaultTesseractPath
	}
	if _, err := os.Stat(bin); err != nil {
		return nil, fmt.Errorf("ocr: tesseract binary not found %q: %w", bin, err)
	}

	if os.Getenv("TESSDATA_PREFIX") == "" {
		derived := filepath.Join(filepath.Dir(bin), "tessdata")
		os.Setenv("TESSDATA_PREFIX", derived)
	}

	// Preprocess: upscale small images for better OCR accuracy; convert to grayscale
	// and add padding to improve edge detection.
	ocrPath := imagePath
	scale := 1.0
	f, err := os.Open(imagePath)
	if err == nil {
		cfg, _, derr := image.DecodeConfig(f)
		f.Close()
		if derr == nil {
			scale = 1.0
			if cfg.Width < 1500 {
				scale = float64(1500) / float64(cfg.Width)
				if scale > 4.0 {
					scale = 4.0
				}
			}
			if scale > 1.0 {
				if upscaled, uerr := preprocessImage(imagePath, scale); uerr == nil {
					ocrPath = upscaled
					defer os.Remove(upscaled)
				} else {
					scale = 1.0
				}
			}
		}
	}

	psm := os.Getenv("TESSERACT_PSM")
	if psm == "" {
		psm = "3"
	}
	configArgs := []string{
		"--oem", "1",
		"--psm", psm,
		"-c", "load_system_dawg=0",
		"-c", "load_freq_dawg=0",
		"-c", "preserve_interword_spaces=1",
	}
	cmd := exec.Command(bin, append([]string{ocrPath, "stdout", "tsv"}, configArgs...)...)
	if os.Getenv("TESSDATA_PREFIX") != "" {
		cmd.Env = append(os.Environ(), "TESSDATA_PREFIX="+os.Getenv("TESSDATA_PREFIX"))
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ocr: tesseract failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	words, err := ParseTSV(bytes.NewReader(stdout.Bytes()))
	if err != nil {
		return nil, err
	}
	if scale != 1.0 {
		for i := range words {
			words[i].X0 = int(float64(words[i].X0) / scale)
			words[i].Y0 = int(float64(words[i].Y0) / scale)
			words[i].X1 = int(float64(words[i].X1) / scale)
			words[i].Y1 = int(float64(words[i].Y1) / scale)
		}
	}
	return words, nil
}

// preprocessImage creates an upscaled, grayscale (with auto-invert for dark
// backgrounds), binarized (Otsu), padded copy of the image for better OCR.
func preprocessImage(src string, factor float64) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}
	b := img.Bounds()

	// Convert to grayscale.
	gray := image.NewGray(b)
	draw.Draw(gray, b, img, b.Min, draw.Src)

	// Auto-invert if the background is dark (mean brightness < 128).
	var sum int64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			sum += int64(gray.GrayAt(x, y).Y)
		}
	}
	pixels := int64(b.Dx() * b.Dy())
	mean := uint8(0)
	if pixels > 0 {
		mean = uint8(sum / pixels)
	}
	if mean < 128 {
		// invert
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				gray.SetGray(x, y, color.Gray{Y: 255 - gray.GrayAt(x, y).Y})
			}
		}
	}

	// Otsu binarization.
	hist := make([]int, 256)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			hist[gray.GrayAt(x, y).Y]++
		}
	}
	total := b.Dx() * b.Dy()
	var sumB int64
	for i := 0; i < 256; i++ {
		sumB += int64(i * hist[i])
	}
	var wB, wF int
	var sumF int64
	maxVar := 0.0
	threshold := 128
	for t := 0; t < 256; t++ {
		wB += hist[t]
		if wB == 0 {
			continue
		}
		wF = total - wB
		if wF == 0 {
			break
		}
		sumB -= int64(t * hist[t]) // approximate; recompute below
		// recompute sums properly
		_ = sumB
		_ = sumF
	}
	// Recompute threshold cleanly.
	var sumTotal int64
	for i := 0; i < 256; i++ {
		sumTotal += int64(i * hist[i])
	}
	wB = 0
	var sumB2 int64
	maxVar = 0
	threshold = 128
	for t := 0; t < 256; t++ {
		wB += hist[t]
		if wB == 0 {
			continue
		}
		wF = total - wB
		if wF == 0 {
			break
		}
		sumB2 += int64(t * hist[t])
		mB := float64(sumB2) / float64(wB)
		mF := float64(sumTotal-sumB2) / float64(wF)
		between := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)
		if between > maxVar {
			maxVar = between
			threshold = t
		}
	}

	bin := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := uint8(255)
			if gray.GrayAt(x, y).Y < uint8(threshold) {
				v = 0
			}
			bin.SetGray(x, y, color.Gray{Y: v})
		}
	}

	// Upscale.
	nw := int(float64(b.Dx())*factor) + 20
	nh := int(float64(b.Dy())*factor) + 20
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			dst.Set(x, y, image.White)
		}
	}
	scaled := image.NewRGBA(image.Rect(0, 0, int(float64(b.Dx())*factor), int(float64(b.Dy())*factor)))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), bin, b, draw.Over, nil)
	draw.Draw(dst, scaled.Bounds().Add(image.Pt(10, 10)), scaled, image.Point{}, draw.Src)

	tmp, err := os.CreateTemp("", "ocrpdf-prep-*.png")
	if err != nil {
		return "", err
	}
	tmp.Close()
	out, err := os.Create(tmp.Name())
	if err != nil {
		return "", err
	}
	defer out.Close()
	if err := png.Encode(out, dst); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}
