package core

import (
	"testing"

	"github.com/framegrace/texelui/color"
	"github.com/gdamore/tcell/v2"
)

func makeCellBuf(w, h int) [][]Cell {
	buf := make([][]Cell, h)
	for i := range buf {
		buf[i] = make([]Cell, w)
	}
	return buf
}

func TestSetDynamicCell_Static(t *testing.T) {
	buf := makeCellBuf(10, 5)
	p := NewPainter(buf, Rect{X: 0, Y: 0, W: 10, H: 5})

	ds := color.DynamicStyle{
		FG: color.Solid(tcell.ColorRed),
		BG: color.Solid(tcell.ColorBlue),
	}
	p.SetDynamicCell(3, 2, 'A', ds)

	c := buf[2][3]
	if c.Ch != 'A' {
		t.Fatalf("expected 'A', got %q", c.Ch)
	}
	fg, bg, _ := c.Style.Decompose()
	if fg != tcell.ColorRed {
		t.Errorf("expected FG red, got %v", fg)
	}
	if bg != tcell.ColorBlue {
		t.Errorf("expected BG blue, got %v", bg)
	}
}

func TestSetDynamicCell_Gradient(t *testing.T) {
	buf := makeCellBuf(10, 1)
	p := NewPainter(buf, Rect{X: 0, Y: 0, W: 10, H: 1})
	p.SetWidgetRect(Rect{X: 0, Y: 0, W: 10, H: 1})

	grad := color.Linear(0, color.Stop(0, tcell.ColorRed), color.Stop(1, tcell.ColorBlue)).WithLocal().Build()
	ds := color.DynamicStyle{
		FG: grad,
		BG: color.Solid(tcell.ColorBlack),
	}

	p.SetDynamicCell(0, 0, 'L', ds)
	p.SetDynamicCell(9, 0, 'R', ds)

	fg0, _, _ := buf[0][0].Style.Decompose()
	fg9, _, _ := buf[0][9].Style.Decompose()

	if fg0 == fg9 {
		t.Errorf("gradient should produce different colors at x=0 and x=9, both got %v", fg0)
	}
}

func TestWithClip_PropagatesContext(t *testing.T) {
	buf := makeCellBuf(20, 20)
	p := NewPainter(buf, Rect{X: 0, Y: 0, W: 20, H: 20})
	p.SetPaneRect(Rect{X: 1, Y: 2, W: 15, H: 15})
	p.SetScreenSize(80, 24)
	p.SetTime(1.5)
	p.SetWidgetRect(Rect{X: 3, Y: 4, W: 10, H: 10})

	sub := p.WithClip(Rect{X: 2, Y: 2, W: 10, H: 10})

	if sub.paneRect != p.paneRect {
		t.Errorf("paneRect not propagated: got %v, want %v", sub.paneRect, p.paneRect)
	}
	if sub.widgetRect != p.widgetRect {
		t.Errorf("widgetRect not propagated: got %v, want %v", sub.widgetRect, p.widgetRect)
	}
	if sub.screenW != 80 || sub.screenH != 24 {
		t.Errorf("screenSize not propagated: got %d×%d, want 80×24", sub.screenW, sub.screenH)
	}
	if sub.time != 1.5 {
		t.Errorf("time not propagated: got %v, want 1.5", sub.time)
	}
	// hasAnim should NOT be propagated
	if sub.hasAnim {
		t.Error("hasAnim should not be propagated to sub-painter")
	}

	// Also test empty-clip path
	empty := p.WithClip(Rect{X: 100, Y: 100, W: 1, H: 1})
	if empty.paneRect != p.paneRect {
		t.Error("paneRect not propagated in empty-clip path")
	}
	if empty.screenW != 80 {
		t.Error("screenW not propagated in empty-clip path")
	}
	if empty.time != 1.5 {
		t.Error("time not propagated in empty-clip path")
	}
}

func TestHasAnimations(t *testing.T) {
	buf := makeCellBuf(10, 5)
	p := NewPainter(buf, Rect{X: 0, Y: 0, W: 10, H: 5})

	if p.HasAnimations() {
		t.Error("expected HasAnimations false initially")
	}

	// Static color should not set hasAnim
	ds := color.DynamicStyle{
		FG: color.Solid(tcell.ColorWhite),
		BG: color.Solid(tcell.ColorBlack),
	}
	p.SetDynamicCell(0, 0, 'X', ds)
	if p.HasAnimations() {
		t.Error("expected HasAnimations false after static draw")
	}

	// AnimatedFunc should set hasAnim
	animated := color.AnimatedFunc(func(ctx color.ColorContext) tcell.Color {
		return tcell.ColorGreen
	})
	dsAnim := color.DynamicStyle{
		FG: animated,
		BG: color.Solid(tcell.ColorBlack),
	}
	p.SetDynamicCell(1, 0, 'Y', dsAnim)
	if !p.HasAnimations() {
		t.Error("expected HasAnimations true after animated draw")
	}
}

func TestFillDynamic(t *testing.T) {
	buf := makeCellBuf(10, 10)
	p := NewPainter(buf, Rect{X: 0, Y: 0, W: 10, H: 10})

	ds := color.DynamicStyle{
		FG: color.Solid(tcell.ColorYellow),
		BG: color.Solid(tcell.ColorBlack),
	}
	p.FillDynamic(Rect{X: 2, Y: 3, W: 3, H: 3}, '#', ds)

	for y := 3; y < 6; y++ {
		for x := 2; x < 5; x++ {
			c := buf[y][x]
			if c.Ch != '#' {
				t.Errorf("cell (%d,%d): expected '#', got %q", x, y, c.Ch)
			}
		}
	}
	// Verify cells outside the fill are untouched
	if buf[0][0].Ch != 0 {
		t.Error("cell (0,0) should be untouched")
	}
}
