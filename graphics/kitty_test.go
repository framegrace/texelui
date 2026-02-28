package graphics

import (
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestKittyCapability(t *testing.T) {
	p := NewKittyProvider()
	if p.Capability() != core.GraphicsKitty {
		t.Errorf("expected GraphicsKitty, got %d", p.Capability())
	}
}

func TestKittyCreateSurfaceStub(t *testing.T) {
	p := NewKittyProvider()
	s := p.CreateSurface(10, 10)
	if s == nil {
		t.Fatal("CreateSurface returned nil")
	}
}
