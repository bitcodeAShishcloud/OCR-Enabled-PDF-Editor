package ocr

import (
	"bytes"
	"encoding/csv"
	"fmt"
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

	cmd := exec.Command(bin, imagePath, "stdout", "tsv")
	if os.Getenv("TESSDATA_PREFIX") != "" {
		cmd.Env = append(os.Environ(), "TESSDATA_PREFIX="+os.Getenv("TESSDATA_PREFIX"))
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ocr: tesseract failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return ParseTSV(bytes.NewReader(stdout.Bytes()))
}