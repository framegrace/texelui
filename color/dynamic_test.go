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
