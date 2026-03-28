// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package color

import (
	"math"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestLinearGradient_Horizontal(t *testing.T) {
	red := tcell.NewRGBColor(255, 0, 0)
	blue := tcell.NewRGBColor(0, 0, 255)
	g := Linear(0, Stop(0, red), Stop(1, blue)).WithLocal().Build()

	ctx0 := ColorContext{X: 0, Y: 0, W: 10, H: 1}
	ctx9 := ColorContext{X: 9, Y: 0, W: 10, H: 1}

	c0 := g.Resolve(ctx0)
	c9 := g.Resolve(ctx9)

	if c0 == c9 {
		t.Error("x=0 and x=9 should produce different colors")
	}

	// x=0 should be red
	r0, g0, b0 := c0.RGB()
	if r0 < 200 || g0 > 50 || b0 > 50 {
		t.Errorf("x=0 should be near red, got RGB(%d,%d,%d)", r0, g0, b0)
	}

	// x=9 should be blue
	r9, g9, b9 := c9.RGB()
	if r9 > 50 || g9 > 50 || b9 < 200 {
		t.Errorf("x=9 should be near blue, got RGB(%d,%d,%d)", r9, g9, b9)
	}
}

func TestLinearGradient_Vertical(t *testing.T) {
	red := tcell.NewRGBColor(255, 0, 0)
	blue := tcell.NewRGBColor(0, 0, 255)
	g := Linear(90, Stop(0, red), Stop(1, blue)).WithLocal().Build()

	ctx0 := ColorContext{X: 0, Y: 0, W: 1, H: 10}
	ctx9 := ColorContext{X: 0, Y: 9, W: 1, H: 10}

	c0 := g.Resolve(ctx0)
	c9 := g.Resolve(ctx9)

	if c0 == c9 {
		t.Error("y=0 and y=9 should produce different colors")
	}

	r0, _, b0 := c0.RGB()
	if r0 < 200 || b0 > 50 {
		t.Errorf("y=0 should be near red, got r=%d b=%d", r0, b0)
	}

	r9, _, b9 := c9.RGB()
	if r9 > 50 || b9 < 200 {
		t.Errorf("y=9 should be near blue, got r=%d b=%d", r9, b9)
	}
}

func TestLinearGradient_ExactStopColor(t *testing.T) {
	red := tcell.NewRGBColor(255, 0, 0)
	blue := tcell.NewRGBColor(0, 0, 255)
	g := Linear(0, Stop(0, red), Stop(1, blue)).WithLocal().Build()

	// At position 0.0, color should be exactly the first stop.
	c := g.Resolve(ColorContext{X: 0, Y: 0, W: 10, H: 10})
	r, gb, b := c.RGB()
	if r != 255 || gb != 0 || b != 0 {
		t.Errorf("position 0.0 should be exactly red, got RGB(%d,%d,%d)", r, gb, b)
	}
}

func TestRadialGradient(t *testing.T) {
	red := tcell.NewRGBColor(255, 0, 0)
	blue := tcell.NewRGBColor(0, 0, 255)
	g := Radial(0.5, 0.5, Stop(0, red), Stop(1, blue)).WithLocal().Build()

	// Center pixel in a 10x10 grid: normalized to ~(0.5, 0.5), distance ~0
	center := g.Resolve(ColorContext{X: 5, Y: 5, W: 11, H: 11})
	// Corner pixel: normalized to (0, 0), distance from (0.5,0.5) = ~0.707
	corner := g.Resolve(ColorContext{X: 0, Y: 0, W: 11, H: 11})

	if center == corner {
		t.Error("center and corner should produce different colors")
	}

	// Center should be near red
	rc, _, bc := center.RGB()
	if rc < 200 || bc > 50 {
		t.Errorf("center should be near red, got r=%d b=%d", rc, bc)
	}
}

func TestLerpHue_ShortestArc(t *testing.T) {
	// 350 -> 10 should go through 0, not through 180
	h := lerpHue(350, 10, 0.5)
	// Midpoint should be 0 (or 360)
	if h > 10 && h < 350 {
		t.Errorf("lerpHue(350, 10, 0.5) = %f, expected near 0/360", h)
	}
	// More precise: should be 0
	if math.Abs(h-0) > 1 && math.Abs(h-360) > 1 {
		t.Errorf("lerpHue(350, 10, 0.5) = %f, expected ~0 or ~360", h)
	}
}

func TestGradient_SingleStop(t *testing.T) {
	green := tcell.NewRGBColor(0, 255, 0)
	g := Linear(0, Stop(0.5, green)).WithLocal().Build()

	// Single stop should return that color everywhere, and be static.
	if !g.IsStatic() {
		t.Error("single-stop gradient should be static")
	}

	c := g.Resolve(ColorContext{X: 0, Y: 0, W: 10, H: 10})
	r, gv, b := c.RGB()
	if r > 10 || gv < 240 || b > 10 {
		t.Errorf("single stop should return green, got RGB(%d,%d,%d)", r, gv, b)
	}
}

func TestGradient_NoStops(t *testing.T) {
	g := Linear(0).Build()

	if !g.IsStatic() {
		t.Error("zero-stop gradient should be static")
	}

	c := g.Resolve(ColorContext{})
	r, gv, b := c.RGB()
	if r != 0 || gv != 0 || b != 0 {
		t.Errorf("zero stops should return black, got RGB(%d,%d,%d)", r, gv, b)
	}
}

func TestGradient_CoordSources(t *testing.T) {
	red := tcell.NewRGBColor(255, 0, 0)
	blue := tcell.NewRGBColor(0, 0, 255)

	localGrad := Linear(0, Stop(0, red), Stop(1, blue)).WithLocal().Build()
	screenGrad := Linear(0, Stop(0, red), Stop(1, blue)).Build()

	// Context where local and screen coords differ.
	ctx := ColorContext{
		X: 0, Y: 0, W: 10, H: 10, // local: x=0 -> t=0
		SX: 9, SY: 0, SW: 10, SH: 10, // screen: x=9 -> t=1
	}

	cLocal := localGrad.Resolve(ctx)
	cScreen := screenGrad.Resolve(ctx)

	if cLocal == cScreen {
		t.Error("WithLocal and default(screen) should produce different results when coords differ")
	}

	// Local should be red (x=0), screen should be blue (sx=9)
	rl, _, bl := cLocal.RGB()
	if rl < 200 || bl > 50 {
		t.Errorf("local should be red, got r=%d b=%d", rl, bl)
	}

	rs, _, bs := cScreen.RGB()
	if rs > 50 || bs < 200 {
		t.Errorf("screen should be blue, got r=%d b=%d", rs, bs)
	}
}

func TestGradient_PaneCoords(t *testing.T) {
	red := tcell.NewRGBColor(255, 0, 0)
	blue := tcell.NewRGBColor(0, 0, 255)

	paneGrad := Linear(0, Stop(0, red), Stop(1, blue)).WithPane().Build()

	ctx := ColorContext{
		X: 5, Y: 0, W: 10, H: 10,
		PX: 0, PY: 0, PW: 10, PH: 10,
		SX: 5, SY: 0, SW: 20, SH: 20,
	}

	c := paneGrad.Resolve(ctx)
	r, _, b := c.RGB()
	// Pane x=0 -> t=0 -> red
	if r < 200 || b > 50 {
		t.Errorf("pane coords should resolve to red at PX=0, got r=%d b=%d", r, b)
	}
}

func TestGradient_MultipleStops(t *testing.T) {
	red := tcell.NewRGBColor(255, 0, 0)
	green := tcell.NewRGBColor(0, 255, 0)
	blue := tcell.NewRGBColor(0, 0, 255)

	g := Linear(0, Stop(0, red), Stop(0.5, green), Stop(1, blue)).WithLocal().Build()

	// At the midpoint stop, should be near green
	ctx := ColorContext{X: 5, Y: 0, W: 11, H: 1}
	c := g.Resolve(ctx)
	_, gv, _ := c.RGB()
	if gv < 200 {
		t.Errorf("midpoint should be near green, got g=%d", gv)
	}
}
