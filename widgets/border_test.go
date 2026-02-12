package widgets

import (
	"testing"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

func TestBorder_BasicDraw(t *testing.T) {
	buf := createTestBuffer(10, 5)
	b := NewBorder()
	b.SetPosition(0, 0)
	b.Resize(10, 5)

	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 10, H: 5})
	b.Draw(p)

	// Top-left corner should be set
	if buf[0][0].Ch == 0 {
		t.Error("expected top-left corner to be drawn")
	}
	// Top-right corner
	if buf[0][9].Ch == 0 {
		t.Error("expected top-right corner to be drawn")
	}
	// Bottom-left corner
	if buf[4][0].Ch == 0 {
		t.Error("expected bottom-left corner to be drawn")
	}
	// Bottom-right corner
	if buf[4][9].Ch == 0 {
		t.Error("expected bottom-right corner to be drawn")
	}
	// Top border middle
	if buf[0][5].Ch != '─' {
		t.Errorf("expected horizontal line on top border, got %q", buf[0][5].Ch)
	}
	// Left border middle
	if buf[2][0].Ch != '│' {
		t.Errorf("expected vertical line on left border, got %q", buf[2][0].Ch)
	}
}

func TestBorder_TitleRendering(t *testing.T) {
	buf := createTestBuffer(20, 5)
	b := NewBorder()
	b.SetPosition(0, 0)
	b.Resize(20, 5)
	b.Title = "Hello"

	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 20, H: 5})
	b.Draw(p)

	// Title should be rendered as " Hello " starting at X=1, Y=0
	// buf[0][1] = ' ', buf[0][2] = 'H', buf[0][3] = 'e', etc.
	expected := " Hello "
	for i, ch := range expected {
		got := buf[0][1+i].Ch
		if got != ch {
			t.Errorf("title position %d: expected %q, got %q", i, ch, got)
		}
	}
}

func TestBorder_TitleTruncation(t *testing.T) {
	buf := createTestBuffer(10, 5)
	b := NewBorder()
	b.SetPosition(0, 0)
	b.Resize(10, 5)
	b.Title = "VeryLongTitle"

	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 10, H: 5})
	b.Draw(p)

	// maxLen = 10 - 4 = 6, so title truncated to "VeryLo"
	// Rendered as " VeryLo " starting at X=1
	expected := " VeryLo "
	for i, ch := range expected {
		got := buf[0][1+i].Ch
		if got != ch {
			t.Errorf("truncated title position %d: expected %q, got %q", i, ch, got)
		}
	}
}

func TestBorder_EmptyTitle(t *testing.T) {
	buf := createTestBuffer(10, 5)
	b := NewBorder()
	b.SetPosition(0, 0)
	b.Resize(10, 5)
	// Title is empty by default

	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 10, H: 5})
	b.Draw(p)

	// Top border should be all horizontal line characters (no title)
	for x := 1; x < 9; x++ {
		if buf[0][x].Ch != '─' {
			t.Errorf("expected horizontal line at x=%d, got %q", x, buf[0][x].Ch)
		}
	}
}

func TestBorder_TitleTooNarrow(t *testing.T) {
	buf := createTestBuffer(4, 3)
	b := NewBorder()
	b.SetPosition(0, 0)
	b.Resize(4, 3)
	b.Title = "Hi"

	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 4, H: 3})
	b.Draw(p)

	// Width 4 means maxLen = 0, so no title should be drawn
	// Top border should be horizontal lines
	if buf[0][1].Ch != '─' && buf[0][2].Ch != '─' {
		t.Error("expected no title drawn when border too narrow")
	}
}

func TestBorder_DetermineStyle_Resizing(t *testing.T) {
	b := NewBorder()
	b.SetPosition(0, 0)
	b.Resize(10, 5)

	// Set custom styles for easy identification
	normalStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	focusedStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	resizingStyle := tcell.StyleDefault.Foreground(tcell.ColorRed)
	b.Style = normalStyle
	b.FocusedStyle = focusedStyle
	b.ResizingStyle = resizingStyle

	// Normal state
	got := b.determineStyle()
	if got != normalStyle {
		t.Error("expected normal style when not focused and not resizing")
	}

	// Focused state
	b.SetFocusable(true)
	b.BaseWidget.Focus()
	got = b.determineStyle()
	if got != focusedStyle {
		t.Error("expected focused style when focused")
	}

	// Resizing takes priority over focused
	b.IsResizing = true
	got = b.determineStyle()
	if got != resizingStyle {
		t.Error("expected resizing style to take priority over focused")
	}

	// Resizing without focus
	b.BaseWidget.Blur()
	got = b.determineStyle()
	if got != resizingStyle {
		t.Error("expected resizing style even when not focused")
	}
}

func TestBorder_ResizingStyleDraw(t *testing.T) {
	buf := createTestBuffer(10, 5)
	b := NewBorder()
	b.SetPosition(0, 0)
	b.Resize(10, 5)

	resizingFG := tcell.NewRGBColor(255, 0, 0)
	b.ResizingStyle = tcell.StyleDefault.Foreground(resizingFG)
	b.IsResizing = true

	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 10, H: 5})
	b.Draw(p)

	// Border cells should use the resizing style
	fg, _, _ := buf[0][5].Style.Decompose()
	if fg != resizingFG {
		t.Errorf("expected resizing FG color, got %v", fg)
	}
}

func TestBorder_ClientRect(t *testing.T) {
	b := NewBorder()
	b.SetPosition(5, 10)
	b.Resize(20, 15)

	cr := b.ClientRect()
	if cr.X != 6 || cr.Y != 11 || cr.W != 18 || cr.H != 13 {
		t.Errorf("expected ClientRect {6,11,18,13}, got {%d,%d,%d,%d}", cr.X, cr.Y, cr.W, cr.H)
	}
}

func TestBorder_TooSmall(t *testing.T) {
	buf := createTestBuffer(1, 1)
	b := NewBorder()
	b.SetPosition(0, 0)
	b.Resize(1, 1)
	b.Title = "X"

	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 1, H: 1})
	b.Draw(p)

	// ClientRect should have zero dimensions
	cr := b.ClientRect()
	if cr.W != 0 || cr.H != 0 {
		t.Errorf("expected zero client rect for 1x1 border, got %dx%d", cr.W, cr.H)
	}
}
