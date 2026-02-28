package core

import (
	"image"
	"testing"
)

func TestGraphicsCapabilityOrdering(t *testing.T) {
	if GraphicsNone >= GraphicsHalfBlock {
		t.Error("None should be less than HalfBlock")
	}
	if GraphicsHalfBlock >= GraphicsKitty {
		t.Error("HalfBlock should be less than Kitty")
	}
}

func TestImageSurfaceInterface(t *testing.T) {
	var _ ImageSurface = (*mockSurface)(nil)
}

type mockSurface struct {
	id  uint32
	buf *image.RGBA
}

func (m *mockSurface) ID() uint32                              { return m.id }
func (m *mockSurface) Buffer() *image.RGBA                     { return m.buf }
func (m *mockSurface) Update() error                           { return nil }
func (m *mockSurface) Place(p *Painter, rect Rect, zIndex int) {}
func (m *mockSurface) Delete()                                 {}

func TestGraphicsProviderV2Interface(t *testing.T) {
	var _ GraphicsProvider = (*mockProviderV2)(nil)
}

type mockProviderV2 struct{}

func (m *mockProviderV2) Capability() GraphicsCapability      { return GraphicsKitty }
func (m *mockProviderV2) CreateSurface(w, h int) ImageSurface { return nil }
func (m *mockProviderV2) Reset()                              {}

func TestPainterGraphicsProvider(t *testing.T) {
	buf := make([][]Cell, 2)
	for i := range buf {
		buf[i] = make([]Cell, 4)
	}

	p := NewPainter(buf, Rect{X: 0, Y: 0, W: 4, H: 2})
	if p.GraphicsProvider() != nil {
		t.Error("expected nil provider by default")
	}

	p2 := NewPainterWithGraphics(buf, Rect{X: 0, Y: 0, W: 4, H: 2}, &mockProviderV2{})
	gp := p2.GraphicsProvider()
	if gp == nil {
		t.Fatal("expected non-nil provider")
	}
	if gp.Capability() != GraphicsKitty {
		t.Errorf("expected Kitty, got %d", gp.Capability())
	}
}

func TestPainterWithClipInheritsGraphicsProvider(t *testing.T) {
	buf := make([][]Cell, 4)
	for i := range buf {
		buf[i] = make([]Cell, 8)
	}

	provider := &mockProviderV2{}
	p := NewPainterWithGraphics(buf, Rect{X: 0, Y: 0, W: 8, H: 4}, provider)

	clipped := p.WithClip(Rect{X: 1, Y: 1, W: 4, H: 2})
	if clipped.GraphicsProvider() != provider {
		t.Error("WithClip should propagate GraphicsProvider")
	}
}
