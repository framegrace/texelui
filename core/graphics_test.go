package core

import "testing"

func TestGraphicsCapabilityOrdering(t *testing.T) {
	if GraphicsNone >= GraphicsHalfBlock {
		t.Error("None should be less than HalfBlock")
	}
	if GraphicsHalfBlock >= GraphicsKitty {
		t.Error("HalfBlock should be less than Kitty")
	}
}

func TestImagePlacementFields(t *testing.T) {
	p := ImagePlacement{
		ID:      42,
		Rect:    Rect{X: 5, Y: 10, W: 20, H: 8},
		ImgData: []byte{0x89, 0x50, 0x4E, 0x47},
		ZIndex:  -1,
	}
	if p.ID != 42 {
		t.Errorf("expected ID 42, got %d", p.ID)
	}
	if p.Rect.W != 20 || p.Rect.H != 8 {
		t.Errorf("unexpected rect: %+v", p.Rect)
	}
	if p.ZIndex != -1 {
		t.Errorf("expected ZIndex -1, got %d", p.ZIndex)
	}
}

func TestPainterGraphicsProvider(t *testing.T) {
	buf := make([][]Cell, 2)
	for i := range buf {
		buf[i] = make([]Cell, 4)
	}

	// Without provider
	p := NewPainter(buf, Rect{X: 0, Y: 0, W: 4, H: 2})
	if p.GraphicsProvider() != nil {
		t.Error("expected nil provider by default")
	}

	// With provider
	p2 := NewPainterWithGraphics(buf, Rect{X: 0, Y: 0, W: 4, H: 2}, &testProvider{cap: GraphicsKitty})
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

	provider := &testProvider{cap: GraphicsKitty}
	p := NewPainterWithGraphics(buf, Rect{X: 0, Y: 0, W: 8, H: 4}, provider)

	clipped := p.WithClip(Rect{X: 1, Y: 1, W: 4, H: 2})
	if clipped.GraphicsProvider() != provider {
		t.Error("WithClip should propagate GraphicsProvider")
	}
}

// testProvider is a minimal GraphicsProvider for testing.
type testProvider struct {
	cap        GraphicsCapability
	placements []ImagePlacement
	deleted    []uint32
	allDeleted bool
}

func (p *testProvider) Capability() GraphicsCapability { return p.cap }
func (p *testProvider) PlaceImage(pl ImagePlacement) error {
	p.placements = append(p.placements, pl)
	return nil
}
func (p *testProvider) DeleteImage(id uint32) { p.deleted = append(p.deleted, id) }
func (p *testProvider) DeleteAll()            { p.allDeleted = true }
