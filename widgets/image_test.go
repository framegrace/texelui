package widgets

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestImage_FromBytes(t *testing.T) {
	// Create a small test image (4x4 red square).
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	imgWidget := NewImage(buf.Bytes(), "red square")
	imgWidget.Resize(4, 2) // 4 cols, 2 rows (2 pixels per row via half-blocks)

	cellBuf := createTestBuffer(4, 2)
	p := core.NewPainter(cellBuf, core.Rect{X: 0, Y: 0, W: 4, H: 2})

	imgWidget.Draw(p)

	// Verify cells are not empty (block art should produce characters).
	nonEmpty := 0
	for _, row := range cellBuf {
		for _, c := range row {
			if c.Ch != 0 && c.Ch != ' ' {
				nonEmpty++
			}
		}
	}
	if nonEmpty == 0 {
		t.Error("expected non-empty block art cells")
	}
}

func TestImage_BlockArtColors(t *testing.T) {
	// Create a 2x2 image: top row red, bottom row blue.
	// With half-blocks, a 2-col x 1-row widget maps to:
	//   fg (top pixel) = red, bg (bottom pixel) = blue.
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{255, 0, 0, 255})
	img.Set(0, 1, color.RGBA{0, 0, 255, 255})
	img.Set(1, 1, color.RGBA{0, 0, 255, 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	imgWidget := NewImage(buf.Bytes(), "red-blue")
	imgWidget.Resize(2, 1)

	cellBuf := createTestBuffer(2, 1)
	p := core.NewPainter(cellBuf, core.Rect{X: 0, Y: 0, W: 2, H: 1})

	imgWidget.Draw(p)

	// Each cell should use the upper-half-block character.
	for x := range 2 {
		if cellBuf[0][x].Ch != '▀' {
			t.Errorf("cell[0][%d].Ch = %q, want '▀'", x, cellBuf[0][x].Ch)
		}
	}
}

func TestImage_AltTextFallback(t *testing.T) {
	// Invalid image data should fall back to alt text.
	imgWidget := NewImage([]byte("not an image"), "fallback text")
	imgWidget.Resize(20, 1)

	cellBuf := createTestBuffer(20, 1)
	p := core.NewPainter(cellBuf, core.Rect{X: 0, Y: 0, W: 20, H: 1})

	imgWidget.Draw(p)

	text := ""
	for _, c := range cellBuf[0] {
		if c.Ch != 0 {
			text += string(c.Ch)
		}
	}
	expected := "[img: fallback text]"
	if text != expected {
		t.Errorf("expected %q, got %q", expected, text)
	}
}

func TestImage_NotFocusable(t *testing.T) {
	imgWidget := NewImage([]byte("data"), "test")
	if imgWidget.Focusable() {
		t.Error("expected Image widget to not be focusable")
	}
}
