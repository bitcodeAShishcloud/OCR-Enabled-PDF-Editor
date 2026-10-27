package ocr

import (
	"os"
	"testing"

	"github.com/fogleman/gg"
)

func TestCLIBackendSmoke(t *testing.T) {
	bin := DefaultTesseractPath
	if v := os.Getenv("TESSERACT_PATH"); v != "" {
		bin = v
	}
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("tesseract binary not found: %v", err)
	}
	fontPath := `C:\Windows\Fonts\arial.ttf`
	if v := os.Getenv("FONT_PATH"); v != "" {
		fontPath = v
	}
	if _, err := os.Stat(fontPath); err != nil {
		t.Skipf("font not found: %v", err)
	}

	dir := t.TempDir()
	imgPath := dir + `\smoke.png`

	dc := gg.NewContext(400, 120)
	dc.SetRGB(1, 1, 1)
	dc.Clear()
	if err := dc.LoadFontFace(fontPath, 24); err != nil {
		t.Fatalf("load font: %v", err)
	}
	dc.SetRGB(0, 0, 0)
	dc.DrawStringAnchored("Hello World", 200, 60, 0.5, 0.5)
	if err := dc.SavePNG(imgPath); err != nil {
		t.Fatalf("save png: %v", err)
	}

	engine := &cliEngine{tesseractPath: bin}
	words, err := engine.Recognize(imgPath)
	if err != nil {
		t.Fatalf("recognize: %v", err)
	}
	if len(words) == 0 {
		t.Fatal("expected at least one word")
	}
	found := false
	for _, w := range words {
		if w.Text != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no word with non-empty text: %+v", words)
	}
}