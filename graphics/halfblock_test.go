package graphics

import (
	"image/color"
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestHalfBlockCreateSurface(t *testing.T) {
	p := NewHalfBlockProvider()
	s := p.CreateSurface(20, 10)
	if s == nil {
		t.Fatal("CreateSurface returned nil")
	}
	if s.ID() == 0 {
		t.Error("expected non-zero ID")
	}
	buf := s.Buffer()
	if buf.Bounds().Dx() != 20 || buf.Bounds().Dy() != 10 {
		t.Errorf("got %dx%d, want 20x10", buf.Bounds().Dx(), buf.Bounds().Dy())
	}
}

func TestHalfBlockSurfaceIDsUnique(t *testing.T) {
	p := NewHalfBlockProvider()
	s1 := p.CreateSurface(10, 10)
	s2 := p.CreateSurface(10, 10)
	if s1.ID() == s2.ID() {
		t.Errorf("surfaces have same ID: %d", s1.ID())
	}
}

func TestHalfBlockCapability(t *testing.T) {
	p := NewHalfBlockProvider()
	if p.Capability() != core.GraphicsHalfBlock {
		t.Errorf("got %d, want GraphicsHalfBlock", p.Capability())
	}
}

func TestHalfBlockPlaceRendersCharacters(t *testing.T) {
	p := NewHalfBlockProvider()
	s := p.CreateSurface(2, 4) // 2px wide, 4px tall -> 2 cols x 2 rows

	// Set pixels: top-left red, below it green
	s.Buffer().Set(0, 0, color.RGBA{255, 0, 0, 255})
	s.Buffer().Set(0, 1, color.RGBA{0, 255, 0, 255})
	s.Buffer().Set(1, 0, color.RGBA{0, 0, 255, 255})
	s.Buffer().Set(1, 1, color.RGBA{255, 255, 0, 255})

	buf := make([][]core.Cell, 24)
	for i := range buf {
		buf[i] = make([]core.Cell, 80)
	}
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 80, H: 24})

	s.Place(painter, core.Rect{X: 5, Y: 10, W: 2, H: 2}, 0)

	// Check that half-block character was placed
	cell := buf[10][5]
	if cell.Ch != '\u2580' {
		t.Errorf("expected upper-half-block at (5,10), got %q", cell.Ch)
	}
}

func TestHalfBlockUpdateIsNoop(t *testing.T) {
	p := NewHalfBlockProvider()
	s := p.CreateSurface(10, 10)
	if err := s.Update(); err != nil {
		t.Errorf("Update returned error: %v", err)
	}
}

func TestHalfBlockResetIsNoop(t *testing.T) {
	p := NewHalfBlockProvider()
	p.Reset() // should not panic
}

func TestHalfBlockDeleteFreesBuffer(t *testing.T) {
	p := NewHalfBlockProvider()
	s := p.CreateSurface(10, 10)
	s.Delete()
	if s.Buffer() != nil {
		t.Error("expected nil buffer after Delete")
	}
}
