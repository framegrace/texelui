// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: texelui/color/dynamic.go
// Summary: Dynamic color types for position/time-dependent color resolution.

package color

import "github.com/gdamore/tcell/v2"

// ColorContext provides spatial and temporal information for color resolution.
type ColorContext struct {
	X, Y   int     // Widget-local coordinates
	W, H   int     // Widget dimensions
	PX, PY int     // Pane coordinates
	PW, PH int     // Pane dimensions
	SX, SY int     // Screen-absolute coordinates
	SW, SH int     // Screen dimensions
	T      float32 // Animation time
}

// ColorFunc computes a color from spatial and temporal context.
type ColorFunc func(ctx ColorContext) tcell.Color

// DynamicColor can be static or a function of position/time.
type DynamicColor struct {
	static   tcell.Color
	fn       ColorFunc
	animated bool
}

// Solid creates a static DynamicColor that always resolves to c.
func Solid(c tcell.Color) DynamicColor {
	return DynamicColor{static: c}
}

// Func creates a spatial DynamicColor that computes its value from context.
func Func(fn ColorFunc) DynamicColor {
	return DynamicColor{fn: fn}
}

// AnimatedFunc creates a time-dependent DynamicColor.
func AnimatedFunc(fn ColorFunc) DynamicColor {
	return DynamicColor{fn: fn, animated: true}
}

// Resolve returns the concrete color for the given context.
// If the color has a function, it is called; otherwise the static value is returned.
func (dc DynamicColor) Resolve(ctx ColorContext) tcell.Color {
	if dc.fn != nil {
		return dc.fn(ctx)
	}
	return dc.static
}

// IsStatic reports whether the color is a fixed value (no function).
func (dc DynamicColor) IsStatic() bool {
	return dc.fn == nil
}

// IsAnimated reports whether the color depends on time.
func (dc DynamicColor) IsAnimated() bool {
	return dc.animated
}

// IsZero reports whether the color was never explicitly set.
func (dc DynamicColor) IsZero() bool {
	return dc.fn == nil && dc.static == 0
}

// DynamicStyle combines dynamic FG/BG with attributes.
type DynamicStyle struct {
	FG    DynamicColor
	BG    DynamicColor
	Attrs tcell.AttrMask
	URL   string
}

// StyleFrom converts a tcell.Style into a DynamicStyle with all static colors.
// Note: tcell.Style does not expose a URL getter, so URL is left empty.
// Set it manually if needed.
func StyleFrom(s tcell.Style) DynamicStyle {
	fg, bg, attrs := s.Decompose()
	return DynamicStyle{
		FG:    Solid(fg),
		BG:    Solid(bg),
		Attrs: attrs,
	}
}
