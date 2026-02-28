package graphics

import (
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestHalfBlockCapability(t *testing.T) {
	p := NewHalfBlockProvider()
	if p.Capability() != core.GraphicsHalfBlock {
		t.Errorf("expected HalfBlock, got %d", p.Capability())
	}
}

func TestHalfBlockCreateSurfaceStub(t *testing.T) {
	p := NewHalfBlockProvider()
	s := p.CreateSurface(10, 10)
	if s == nil {
		t.Fatal("CreateSurface returned nil")
	}
}
