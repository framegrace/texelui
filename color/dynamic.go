// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: texelui/color/dynamic.go
// Summary: Dynamic color types for position/time-dependent color resolution.

package color

import (
	"math"

	"github.com/gdamore/tcell/v2"
)

// ColorContext provides spatial and temporal information for color resolution.
type ColorContext struct {
	X, Y   int     // Widget-local coordinates
	W, H   int     // Widget dimensions
	PX, PY int     // Pane coordinates
	PW, PH int     // Pane dimensions
	SX, SY int     // Screen-absolute coordinates
	SW, SH int     // Screen dimensions
	T      float32 // Animation time in seconds (accumulated from deltas)
	DT     float32 // Delta time this frame in seconds (0 for data-driven renders)
}

// ColorFunc computes a color from spatial and temporal context.
type ColorFunc func(ctx ColorContext) tcell.Color

// DynamicColorDesc is a serializable descriptor for a DynamicColor.
// It captures the animation intent so it can travel across the protocol
// and be reconstructed on the client side.
type DynamicColorDesc struct {
	Type   uint8              // DescTypeNone, DescTypeSolid, DescTypePulse, DescTypeFade, DescTypeLinearGrad, DescTypeRadialGrad
	Base   uint32             // RGB packed base color
	Target uint32             // RGB packed target (for fade)
	Easing uint8              // index into EasingByIndex table
	Speed  float32            // oscillations/sec (pulse) or duration in seconds (fade)
	Min    float32            // min scale factor (pulse)
	Max    float32            // max scale factor (pulse)
	Stops  []GradientStopDesc // nil for non-gradient types
}

const (
	DescTypeNone       uint8 = 0
	DescTypeSolid      uint8 = 1
	DescTypePulse      uint8 = 2
	DescTypeFade       uint8 = 3
	DescTypeLinearGrad uint8 = 4
	DescTypeRadialGrad uint8 = 5
)

// GradientStopDesc describes a single stop in a gradient descriptor.
type GradientStopDesc struct {
	Position float32
	Color    DynamicColorDesc // Solid, Pulse, or Fade — no nested gradients
}

// IsAnimated returns true if this descriptor represents an animated color.
func (d DynamicColorDesc) IsAnimated() bool {
	if d.Type >= DescTypePulse && d.Type <= DescTypeFade {
		return true
	}
	for _, s := range d.Stops {
		if s.Color.IsAnimated() {
			return true
		}
	}
	return false
}

// IsGradient returns true if this descriptor represents a gradient.
func (d DynamicColorDesc) IsGradient() bool {
	return d.Type == DescTypeLinearGrad || d.Type == DescTypeRadialGrad
}

// IsDynamic returns true if this descriptor is non-trivial (not None or Solid).
func (d DynamicColorDesc) IsDynamic() bool {
	return d.Type >= DescTypePulse
}

// PackRGB packs r, g, b int32 components into a uint32.
func PackRGB(r, g, b int32) uint32 {
	return (uint32(r)&0xFF)<<16 | (uint32(g)&0xFF)<<8 | uint32(b)&0xFF
}

// UnpackRGB unpacks a uint32 into r, g, b int32 components.
func UnpackRGB(rgb uint32) (int32, int32, int32) {
	return int32((rgb >> 16) & 0xFF), int32((rgb >> 8) & 0xFF), int32(rgb & 0xFF)
}

// DynamicColor can be static or a function of position/time.
type DynamicColor struct {
	static   tcell.Color
	fn       ColorFunc
	animated bool
	desc     DynamicColorDesc
}

// Solid creates a static DynamicColor that always resolves to c.
func Solid(c tcell.Color) DynamicColor {
	r, g, b := c.RGB()
	return DynamicColor{
		static: c,
		desc:   DynamicColorDesc{Type: DescTypeSolid, Base: PackRGB(r, g, b)},
	}
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

// Describe returns a serializable descriptor for this DynamicColor.
// Raw Func closures return DescTypeNone (cannot be serialized).
func (dc DynamicColor) Describe() DynamicColorDesc {
	return dc.desc
}

// Pulse creates an oscillating brightness DynamicColor.
// Base color oscillates between min and max scale factors at speedHz oscillations/sec.
func Pulse(base tcell.Color, min, max, speedHz float32) DynamicColor {
	r, g, b := base.RGB()
	desc := DynamicColorDesc{
		Type:  DescTypePulse,
		Base:  PackRGB(r, g, b),
		Speed: speedHz,
		Min:   min,
		Max:   max,
	}
	fr, fg, fb := float64(r), float64(g), float64(b)
	return DynamicColor{
		static: base,
		fn: func(ctx ColorContext) tcell.Color {
			mid := float64(min+max) / 2
			amp := float64(max-min) / 2
			factor := mid + amp*math.Sin(float64(ctx.T)*float64(speedHz))
			return tcell.NewRGBColor(
				clamp8(int32(fr*factor)),
				clamp8(int32(fg*factor)),
				clamp8(int32(fb*factor)),
			)
		},
		animated: true,
		desc:     desc,
	}
}

// Fade creates a one-shot color transition DynamicColor.
func Fade(from, to tcell.Color, easing string, durationSec float32) DynamicColor {
	fr, ffg, fb := from.RGB()
	tr, tg, tb := to.RGB()
	desc := DynamicColorDesc{
		Type:   DescTypeFade,
		Base:   PackRGB(fr, ffg, fb),
		Target: PackRGB(tr, tg, tb),
		Easing: LookupEasingByName(easing),
		Speed:  durationSec,
	}
	easeFn := LookupEasing(desc.Easing)
	return DynamicColor{
		static: from,
		fn: func(ctx ColorContext) tcell.Color {
			progress := ctx.T / float32(durationSec)
			if progress >= 1 {
				return to
			}
			if progress < 0 {
				return from
			}
			t := easeFn(progress)
			blend := func(a, b int32) int32 {
				return a + int32(float32(b-a)*t)
			}
			return tcell.NewRGBColor(blend(fr, tr), blend(ffg, tg), blend(fb, tb))
		},
		animated: true,
		desc:     desc,
	}
}

// FromDesc reconstructs a DynamicColor from a serializable descriptor.
func FromDesc(d DynamicColorDesc) DynamicColor {
	switch d.Type {
	case DescTypeSolid:
		r, g, b := UnpackRGB(d.Base)
		return Solid(tcell.NewRGBColor(r, g, b))
	case DescTypePulse:
		r, g, b := UnpackRGB(d.Base)
		return Pulse(tcell.NewRGBColor(r, g, b), d.Min, d.Max, d.Speed)
	case DescTypeFade:
		fr, ffg, fb := UnpackRGB(d.Base)
		tr, tg, tb := UnpackRGB(d.Target)
		easingName := "linear"
		for name, idx := range EasingByName {
			if idx == d.Easing {
				easingName = name
				break
			}
		}
		return Fade(tcell.NewRGBColor(fr, ffg, fb), tcell.NewRGBColor(tr, tg, tb), easingName, d.Speed)
	case DescTypeLinearGrad:
		angleDeg := math.Float32frombits(d.Base)
		source := coordSource(d.Easing)
		stops := make([]ColorStop, len(d.Stops))
		for i, sd := range d.Stops {
			stops[i] = ColorStop{
				Position: sd.Position,
				Dynamic:  FromDesc(sd.Color),
			}
		}
		return Linear(angleDeg, stops...).withSource(source).Build()

	case DescTypeRadialGrad:
		cx := d.Speed
		cy := math.Float32frombits(d.Target)
		source := coordSource(d.Easing)
		stops := make([]ColorStop, len(d.Stops))
		for i, sd := range d.Stops {
			stops[i] = ColorStop{
				Position: sd.Position,
				Dynamic:  FromDesc(sd.Color),
			}
		}
		return Radial(cx, cy, stops...).withSource(source).Build()

	default:
		return DynamicColor{}
	}
}

func clamp8(v int32) int32 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
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
