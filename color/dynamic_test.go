// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package color

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestSolid_Resolve(t *testing.T) {
	c := tcell.NewRGBColor(255, 0, 0)
	dc := Solid(c)

	got := dc.Resolve(ColorContext{})
	if got != c {
		t.Errorf("Resolve() = %v, want %v", got, c)
	}
	if !dc.IsStatic() {
		t.Error("IsStatic() = false, want true")
	}
	if dc.IsAnimated() {
		t.Error("IsAnimated() = true, want false")
	}
}

func TestFunc_Resolve(t *testing.T) {
	fn := func(ctx ColorContext) tcell.Color {
		if ctx.X > 5 {
			return tcell.ColorRed
		}
		return tcell.ColorBlue
	}
	dc := Func(fn)

	if dc.IsStatic() {
		t.Error("IsStatic() = true, want false")
	}
	if dc.IsAnimated() {
		t.Error("IsAnimated() = true, want false")
	}

	got := dc.Resolve(ColorContext{X: 10})
	if got != tcell.ColorRed {
		t.Errorf("Resolve(X=10) = %v, want ColorRed", got)
	}

	got = dc.Resolve(ColorContext{X: 2})
	if got != tcell.ColorBlue {
		t.Errorf("Resolve(X=2) = %v, want ColorBlue", got)
	}
}

func TestAnimatedFunc(t *testing.T) {
	fn := func(ctx ColorContext) tcell.Color {
		if ctx.T > 0.5 {
			return tcell.ColorGreen
		}
		return tcell.ColorYellow
	}
	dc := AnimatedFunc(fn)

	if dc.IsStatic() {
		t.Error("IsStatic() = true, want false")
	}
	if !dc.IsAnimated() {
		t.Error("IsAnimated() = false, want true")
	}

	got := dc.Resolve(ColorContext{T: 1.0})
	if got != tcell.ColorGreen {
		t.Errorf("Resolve(T=1.0) = %v, want ColorGreen", got)
	}

	got = dc.Resolve(ColorContext{T: 0.1})
	if got != tcell.ColorYellow {
		t.Errorf("Resolve(T=0.1) = %v, want ColorYellow", got)
	}
}

func TestStyleFrom(t *testing.T) {
	s := tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorNavy).
		Bold(true).
		Underline(true)

	ds := StyleFrom(s)

	fg := ds.FG.Resolve(ColorContext{})
	if fg != tcell.ColorWhite {
		t.Errorf("FG = %v, want ColorWhite", fg)
	}

	bg := ds.BG.Resolve(ColorContext{})
	if bg != tcell.ColorNavy {
		t.Errorf("BG = %v, want ColorNavy", bg)
	}

	wantAttrs := tcell.AttrBold | tcell.AttrUnderline
	if ds.Attrs != wantAttrs {
		t.Errorf("Attrs = %v, want %v", ds.Attrs, wantAttrs)
	}

	if !ds.FG.IsStatic() || !ds.BG.IsStatic() {
		t.Error("StyleFrom should produce static colors")
	}

	// URL field can be set manually (tcell has no URL getter).
	ds.URL = "https://example.com"
	if ds.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", ds.URL, "https://example.com")
	}
}

func TestDynamicColorDesc_Solid(t *testing.T) {
	dc := Solid(tcell.NewRGBColor(100, 150, 200))
	desc := dc.Describe()
	if desc.Type != DescTypeSolid {
		t.Fatalf("expected DescTypeSolid, got %d", desc.Type)
	}
	if desc.Base != packRGBTest(100, 150, 200) {
		t.Errorf("base color mismatch: got %x", desc.Base)
	}
}

func TestDynamicColorDesc_Pulse(t *testing.T) {
	dc := Pulse(tcell.NewRGBColor(100, 150, 200), 0.7, 1.0, 6)
	desc := dc.Describe()
	if desc.Type != DescTypePulse {
		t.Fatalf("expected DescTypePulse, got %d", desc.Type)
	}
	if desc.Min != 0.7 || desc.Max != 1.0 || desc.Speed != 6 {
		t.Errorf("pulse params: min=%.1f max=%.1f speed=%.1f", desc.Min, desc.Max, desc.Speed)
	}
	if !dc.IsAnimated() {
		t.Error("Pulse should be animated")
	}
}

func TestDynamicColorDesc_Fade(t *testing.T) {
	dc := Fade(tcell.NewRGBColor(255, 0, 0), tcell.NewRGBColor(0, 0, 255), "smoothstep", 0.5)
	desc := dc.Describe()
	if desc.Type != DescTypeFade {
		t.Fatalf("expected DescTypeFade, got %d", desc.Type)
	}
	if desc.Base != packRGBTest(255, 0, 0) {
		t.Errorf("base color mismatch")
	}
	if desc.Target != packRGBTest(0, 0, 255) {
		t.Errorf("target color mismatch")
	}
}

func TestDynamicColorDesc_RoundTrip(t *testing.T) {
	original := Pulse(tcell.NewRGBColor(100, 150, 200), 0.7, 1.0, 6)
	desc := original.Describe()
	reconstructed := FromDesc(desc)

	ctx := ColorContext{T: 0.5}
	origColor := original.Resolve(ctx)
	reconColor := reconstructed.Resolve(ctx)
	if origColor != reconColor {
		t.Errorf("round-trip mismatch: original=%v reconstructed=%v", origColor, reconColor)
	}
}

func TestDynamicColorDesc_FuncNotSerializable(t *testing.T) {
	dc := Func(func(ctx ColorContext) tcell.Color { return tcell.ColorRed })
	desc := dc.Describe()
	if desc.Type != DescTypeNone {
		t.Errorf("expected DescTypeNone for raw Func, got %d", desc.Type)
	}
}

func TestDynamicColorDesc_FadeRoundTrip(t *testing.T) {
	original := Fade(tcell.NewRGBColor(255, 0, 0), tcell.NewRGBColor(0, 0, 255), "smoothstep", 2.0)
	desc := original.Describe()
	reconstructed := FromDesc(desc)

	for _, tVal := range []float32{0.0, 0.25, 0.5, 0.75, 1.0} {
		ctx := ColorContext{T: tVal}
		origColor := original.Resolve(ctx)
		reconColor := reconstructed.Resolve(ctx)
		if origColor != reconColor {
			t.Errorf("T=%.2f: round-trip mismatch: original=%v reconstructed=%v", tVal, origColor, reconColor)
		}
	}
}

func TestDynamicColorDesc_LinearGradient(t *testing.T) {
	grad := Linear(0, Stop(0, tcell.NewRGBColor(255, 0, 0)), Stop(1, tcell.NewRGBColor(0, 0, 255))).WithLocal().Build()
	desc := grad.Describe()
	if desc.Type != DescTypeLinearGrad {
		t.Fatalf("expected DescTypeLinearGrad, got %d", desc.Type)
	}
	if len(desc.Stops) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(desc.Stops))
	}
	if desc.Stops[0].Color.Type != DescTypeSolid {
		t.Errorf("stop 0 should be solid, got %d", desc.Stops[0].Color.Type)
	}
}

func TestDynamicColorDesc_LinearGradientRoundTrip(t *testing.T) {
	original := Linear(45, Stop(0, tcell.NewRGBColor(255, 0, 0)), Stop(1, tcell.NewRGBColor(0, 0, 255))).WithLocal().Build()
	desc := original.Describe()
	reconstructed := FromDesc(desc)

	ctx := ColorContext{X: 5, Y: 0, W: 10, H: 1}
	origColor := original.Resolve(ctx)
	reconColor := reconstructed.Resolve(ctx)
	if origColor != reconColor {
		t.Errorf("round-trip mismatch at X=5: original=%v reconstructed=%v", origColor, reconColor)
	}
}

func TestDynamicColorDesc_GradientWithPulseStop(t *testing.T) {
	pulse := Pulse(tcell.NewRGBColor(137, 180, 250), 0.7, 1.0, 6)
	grad := Linear(0, DynStop(0, pulse), Stop(1, tcell.NewRGBColor(30, 30, 46))).WithLocal().Build()
	desc := grad.Describe()
	if desc.Type != DescTypeLinearGrad {
		t.Fatalf("expected DescTypeLinearGrad, got %d", desc.Type)
	}
	if desc.Stops[0].Color.Type != DescTypePulse {
		t.Errorf("stop 0 should be pulse, got %d", desc.Stops[0].Color.Type)
	}
	if !desc.IsAnimated() {
		t.Error("gradient with Pulse stop should be animated")
	}

	// Round-trip
	reconstructed := FromDesc(desc)
	ctx := ColorContext{X: 0, Y: 0, W: 10, H: 1, T: 0.5}
	origColor := grad.Resolve(ctx)
	reconColor := reconstructed.Resolve(ctx)
	if origColor != reconColor {
		t.Errorf("round-trip mismatch: original=%v reconstructed=%v", origColor, reconColor)
	}
}

func TestDynamicColorDesc_RadialGradientRoundTrip(t *testing.T) {
	original := Radial(0.5, 0.5, Stop(0, tcell.NewRGBColor(255, 255, 200)), Stop(1, tcell.NewRGBColor(20, 20, 80))).WithLocal().Build()
	desc := original.Describe()
	if desc.Type != DescTypeRadialGrad {
		t.Fatalf("expected DescTypeRadialGrad, got %d", desc.Type)
	}
	reconstructed := FromDesc(desc)

	ctx := ColorContext{X: 5, Y: 5, W: 10, H: 10}
	origColor := original.Resolve(ctx)
	reconColor := reconstructed.Resolve(ctx)
	if origColor != reconColor {
		t.Errorf("round-trip mismatch: original=%v reconstructed=%v", origColor, reconColor)
	}
}

func packRGBTest(r, g, b int32) uint32 {
	return (uint32(r)&0xFF)<<16 | (uint32(g)&0xFF)<<8 | uint32(b)&0xFF
}
