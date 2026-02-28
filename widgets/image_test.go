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

	// With a HalfBlock-capable provider, the surface Place stub is a no-op,
	// so we test alt-text fallback with no provider.
	cellBuf := createTestBuffer(4, 2)
	p := core.NewPainter(cellBuf, core.Rect{X: 0, Y: 0, W: 4, H: 2})

	imgWidget.Draw(p)

	// With nil provider, capability < HalfBlock, so alt text is rendered.
	text := ""
	for _, c := range cellBuf[0] {
		if c.Ch != 0 {
			text += string(c.Ch)
		}
	}
	if text == "" {
		t.Error("expected non-empty output")
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

func TestImage_SurfacePath(t *testing.T) {
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
	imgWidget.Resize(4, 2)
	imgWidget.SetPosition(5, 10)

	provider := &mockGraphicsProvider{}

	cellBuf := createTestBuffer(20, 15)
	p := core.NewPainterWithGraphics(cellBuf, core.Rect{X: 0, Y: 0, W: 20, H: 15}, provider)

	// Should not panic; surface stub Place is a no-op
	imgWidget.Draw(p)

	if provider.createCount == 0 {
		t.Error("expected CreateSurface to be called")
	}
}

func TestImage_NilProvider_AltText(t *testing.T) {
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
	imgWidget.Resize(20, 1)

	// No provider (nil) - capability < HalfBlock, falls back to alt text
	cellBuf := createTestBuffer(20, 1)
	p := core.NewPainter(cellBuf, core.Rect{X: 0, Y: 0, W: 20, H: 1})

	imgWidget.Draw(p)

	text := ""
	for _, c := range cellBuf[0] {
		if c.Ch != 0 {
			text += string(c.Ch)
		}
	}
	if text == "" {
		t.Error("expected alt text output with nil provider")
	}
}

type mockGraphicsProvider struct {
	createCount int
}

func (p *mockGraphicsProvider) Capability() core.GraphicsCapability {
	return core.GraphicsHalfBlock
}

func (p *mockGraphicsProvider) CreateSurface(w, h int) core.ImageSurface {
	p.createCount++
	return &mockImageSurface{buf: image.NewRGBA(image.Rect(0, 0, w, h))}
}

func (p *mockGraphicsProvider) Reset() {}

type mockImageSurface struct {
	buf *image.RGBA
}

func (s *mockImageSurface) ID() uint32          { return 1 }
func (s *mockImageSurface) Buffer() *image.RGBA { return s.buf }
func (s *mockImageSurface) Update() error       { return nil }
func (s *mockImageSurface) Place(p *core.Painter, rect core.Rect, zIndex int) {}
func (s *mockImageSurface) Delete()             {}
