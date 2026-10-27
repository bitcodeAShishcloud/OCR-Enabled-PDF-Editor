package ocr

import (
	"os"
	"strings"
	"testing"
)

func TestParseTSVBasic(t *testing.T) {
	f, err := os.Open("testdata/sample.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	words, err := ParseTSV(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 3 {
		t.Fatalf("got %d words, want 3", len(words))
	}
	if words[0].Text != "Hello" || words[0].X0 != 89 || words[0].Y0 != 49 ||
		words[0].X1 != 158 || words[0].Y1 != 72 {
		t.Fatalf("unexpected bbox: %+v", words[0])
	}
}

func TestParseTSVIgnoresHeaderAndMinusOne(t *testing.T) {
	f, err := os.Open("testdata/sample.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	words, err := ParseTSV(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 3 {
		t.Fatalf("got %d words, want 3", len(words))
	}
	for _, w := range words {
		if w.Confidence < 0 {
			t.Fatalf("negative confidence word: %+v", w)
		}
		if w.Text == "level" {
			t.Fatalf("header leaked as word: %+v", w)
		}
	}
}

func TestParseTSVVariableFields(t *testing.T) {
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"5\t1\t1\t1\t1\t1\t10\t20\t30\t40\t90.0\tHi\n" +
		"5\t1\t1\t1\t1\t2\t10\t20\t30\t40\t90.0\n"
	r := strings.NewReader(tsv)
	words, err := ParseTSV(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 1 {
		t.Fatalf("got %d words, want 1", len(words))
	}
	if words[0].Text != "Hi" {
		t.Fatalf("got text %q, want Hi", words[0].Text)
	}
}

func TestParseTSVEmptyTextSkipped(t *testing.T) {
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
		"5\t1\t1\t1\t1\t1\t10\t20\t30\t40\t90.0\t\n"
	r := strings.NewReader(tsv)
	words, err := ParseTSV(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 0 {
		t.Fatalf("got %d words, want 0", len(words))
	}
}

func TestParseTSVEmptyInput(t *testing.T) {
	words, err := ParseTSV(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 0 {
		t.Fatalf("got %d words, want 0", len(words))
	}
}