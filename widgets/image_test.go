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

func TestImage_KittyPath(t *testing.T) {
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
	pngData := buf.Bytes()

	imgWidget := NewImage(pngData, "red square")
	imgWidget.Resize(4, 2)
	imgWidget.SetPosition(5, 10)

	provider := &mockGraphicsProvider{cap: core.GraphicsKitty}

	cellBuf := createTestBuffer(20, 15)
	p := core.NewPainterWithGraphics(cellBuf, core.Rect{X: 0, Y: 0, W: 20, H: 15}, provider)

	imgWidget.Draw(p)

	if len(provider.placements) != 1 {
		t.Fatalf("expected 1 placement, got %d", len(provider.placements))
	}

	pl := provider.placements[0]
	if pl.Rect.X != 5 || pl.Rect.Y != 10 {
		t.Errorf("expected position (5,10), got (%d,%d)", pl.Rect.X, pl.Rect.Y)
	}
	if pl.Rect.W != 4 || pl.Rect.H != 2 {
		t.Errorf("expected size (4,2), got (%d,%d)", pl.Rect.W, pl.Rect.H)
	}
	if len(pl.ImgData) == 0 {
		t.Error("expected non-empty image data")
	}

	// Cell buffer should be filled with spaces (placeholder)
	for y := 10; y < 12; y++ {
		for x := 5; x < 9; x++ {
			if cellBuf[y][x].Ch != ' ' {
				t.Errorf("expected space at (%d,%d), got %q", x, y, cellBuf[y][x].Ch)
			}
		}
	}
}

func TestImage_FallbackToHalfBlock(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{0, 255, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	imgWidget := NewImage(buf.Bytes(), "green square")
	imgWidget.Resize(4, 2)

	provider := &mockGraphicsProvider{cap: core.GraphicsHalfBlock}
	cellBuf := createTestBuffer(4, 2)
	p := core.NewPainterWithGraphics(cellBuf, core.Rect{X: 0, Y: 0, W: 4, H: 2}, provider)

	imgWidget.Draw(p)

	if len(provider.placements) != 0 {
		t.Errorf("expected 0 placements for HalfBlock, got %d", len(provider.placements))
	}

	if cellBuf[0][0].Ch != '\u2580' {
		t.Errorf("expected half-block character, got %q", cellBuf[0][0].Ch)
	}
}

func TestImage_NilProvider_UsesHalfBlock(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	imgWidget := NewImage(buf.Bytes(), "blue square")
	imgWidget.Resize(4, 2)

	// No provider (nil) - should use half-block
	cellBuf := createTestBuffer(4, 2)
	p := core.NewPainter(cellBuf, core.Rect{X: 0, Y: 0, W: 4, H: 2})

	imgWidget.Draw(p)

	if cellBuf[0][0].Ch != '\u2580' {
		t.Errorf("expected half-block character with nil provider, got %q", cellBuf[0][0].Ch)
	}
}

type mockGraphicsProvider struct {
	cap        core.GraphicsCapability
	placements []core.ImagePlacement
}

func (p *mockGraphicsProvider) Capability() core.GraphicsCapability { return p.cap }
func (p *mockGraphicsProvider) PlaceImage(pl core.ImagePlacement) error {
	p.placements = append(p.placements, pl)
	return nil
}
func (p *mockGraphicsProvider) DeleteImage(uint32) {}
func (p *mockGraphicsProvider) DeleteAll()         {}
