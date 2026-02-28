package graphics

import (
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestHalfBlockProvider(t *testing.T) {
	p := NewHalfBlockProvider()
	if p.Capability() != core.GraphicsHalfBlock {
		t.Errorf("expected HalfBlock, got %d", p.Capability())
	}
	// PlaceImage is a no-op, should not error
	err := p.PlaceImage(core.ImagePlacement{ID: 1})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Delete methods are no-ops, should not panic
	p.DeleteImage(1)
	p.DeleteAll()
}
