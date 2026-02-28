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
