package widgets

import (
	"testing"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

func TestBufferWidget_BasicDraw(t *testing.T) {
	// Create a source buffer with known content
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	src := [][]core.Cell{
		{{Ch: 'A', Style: style}, {Ch: 'B', Style: style}},
		{{Ch: 'C', Style: style}, {Ch: 'D', Style: style}},
	}

	w := NewBufferWidget(src)
	w.SetPosition(1, 1)
	w.Resize(2, 2)

	// Draw into a larger destination buffer
	dst := createTestBuffer(5, 5)
	p := core.NewPainter(dst, core.Rect{X: 0, Y: 0, W: 5, H: 5})
	w.Draw(p)

	// Cells should be at offset (1,1) in destination
	if dst[1][1].Ch != 'A' {
		t.Errorf("expected 'A' at (1,1), got %q", dst[1][1].Ch)
	}
	if dst[1][2].Ch != 'B' {
		t.Errorf("expected 'B' at (1,2), got %q", dst[1][2].Ch)
	}
	if dst[2][1].Ch != 'C' {
		t.Errorf("expected 'C' at (2,1), got %q", dst[2][1].Ch)
	}
	if dst[2][2].Ch != 'D' {
		t.Errorf("expected 'D' at (2,2), got %q", dst[2][2].Ch)
	}

	// Surrounding cells should be untouched (zero value)
	if dst[0][0].Ch != 0 {
		t.Errorf("expected untouched cell at (0,0), got %q", dst[0][0].Ch)
	}
}

func TestBufferWidget_NilBuffer(t *testing.T) {
	w := NewBufferWidget(nil)
	w.SetPosition(0, 0)
	w.Resize(5, 5)

	dst := createTestBuffer(5, 5)
	p := core.NewPainter(dst, core.Rect{X: 0, Y: 0, W: 5, H: 5})

	// Should not panic
	w.Draw(p)

	// All cells should be untouched
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			if dst[y][x].Ch != 0 {
				t.Errorf("expected untouched cell at (%d,%d), got %q", x, y, dst[y][x].Ch)
			}
		}
	}
}

func TestBufferWidget_EmptyBuffer(t *testing.T) {
	w := NewBufferWidget([][]core.Cell{})
	w.SetPosition(0, 0)
	w.Resize(5, 5)

	dst := createTestBuffer(5, 5)
	p := core.NewPainter(dst, core.Rect{X: 0, Y: 0, W: 5, H: 5})
	w.Draw(p)

	// All cells should be untouched
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			if dst[y][x].Ch != 0 {
				t.Errorf("expected untouched cell at (%d,%d)", x, y)
			}
		}
	}
}

func TestBufferWidget_SmallerThanWidget(t *testing.T) {
	// Buffer is 2x2 but widget is 4x4
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	src := [][]core.Cell{
		{{Ch: 'X', Style: style}, {Ch: 'Y', Style: style}},
	}

	w := NewBufferWidget(src)
	w.SetPosition(0, 0)
	w.Resize(4, 4)

	dst := createTestBuffer(5, 5)
	p := core.NewPainter(dst, core.Rect{X: 0, Y: 0, W: 5, H: 5})
	w.Draw(p)

	// Only the source cells should be drawn
	if dst[0][0].Ch != 'X' {
		t.Errorf("expected 'X' at (0,0), got %q", dst[0][0].Ch)
	}
	if dst[0][1].Ch != 'Y' {
		t.Errorf("expected 'Y' at (0,1), got %q", dst[0][1].Ch)
	}
	// Beyond buffer should be untouched
	if dst[1][0].Ch != 0 {
		t.Errorf("expected untouched at (0,1), got %q", dst[1][0].Ch)
	}
}

func TestBufferWidget_AutoSize(t *testing.T) {
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	src := [][]core.Cell{
		{{Ch: 'A', Style: style}, {Ch: 'B', Style: style}, {Ch: 'C', Style: style}},
		{{Ch: 'D', Style: style}, {Ch: 'E', Style: style}, {Ch: 'F', Style: style}},
	}

	w := NewBufferWidget(src)
	ww, wh := w.Size()
	if ww != 3 || wh != 2 {
		t.Errorf("expected auto-size 3x2, got %dx%d", ww, wh)
	}
}

func TestBufferWidget_SetBuffer(t *testing.T) {
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	src1 := [][]core.Cell{
		{{Ch: 'A', Style: style}},
	}
	src2 := [][]core.Cell{
		{{Ch: 'Z', Style: style}},
	}

	w := NewBufferWidget(src1)
	w.SetPosition(0, 0)
	w.Resize(1, 1)

	dst := createTestBuffer(2, 2)
	p := core.NewPainter(dst, core.Rect{X: 0, Y: 0, W: 2, H: 2})

	w.Draw(p)
	if dst[0][0].Ch != 'A' {
		t.Errorf("expected 'A', got %q", dst[0][0].Ch)
	}

	// Update buffer and redraw
	w.SetBuffer(src2)
	dst = createTestBuffer(2, 2)
	p = core.NewPainter(dst, core.Rect{X: 0, Y: 0, W: 2, H: 2})
	w.Draw(p)
	if dst[0][0].Ch != 'Z' {
		t.Errorf("expected 'Z' after SetBuffer, got %q", dst[0][0].Ch)
	}
}

func TestBufferWidget_InsideBorder(t *testing.T) {
	// Integration test: BufferWidget as child of Border
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	src := [][]core.Cell{
		{{Ch: 'H', Style: style}, {Ch: 'i', Style: style}},
	}

	border := NewBorder()
	border.SetPosition(0, 0)
	border.Resize(6, 3)
	border.Title = "T"

	child := NewBufferWidget(src)
	border.SetChild(child)

	dst := createTestBuffer(6, 3)
	p := core.NewPainter(dst, core.Rect{X: 0, Y: 0, W: 6, H: 3})
	border.Draw(p)

	// Border should be drawn
	if dst[0][0].Ch == 0 {
		t.Error("expected border corner at (0,0)")
	}

	// Content should be inside at (1,1)
	if dst[1][1].Ch != 'H' {
		t.Errorf("expected 'H' at (1,1) inside border, got %q", dst[1][1].Ch)
	}
	if dst[1][2].Ch != 'i' {
		t.Errorf("expected 'i' at (1,2) inside border, got %q", dst[1][2].Ch)
	}
}
