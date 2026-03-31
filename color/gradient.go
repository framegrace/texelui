// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: texelui/color/gradient.go
// Summary: Linear and radial gradient builders with OKLCH interpolation.

package color

import (
	"math"
	"sort"

	"github.com/gdamore/tcell/v2"
)

// ColorStop defines a color at a specific position along a gradient.
type ColorStop struct {
	Color    tcell.Color
	Dynamic  DynamicColor // if set, resolved at draw time instead of Color
	Position float32      // 0.0 to 1.0
}

// Stop creates a ColorStop at the given position with the given color.
func Stop(position float32, c tcell.Color) ColorStop {
	return ColorStop{Color: c, Position: position}
}

// DynStop creates a ColorStop from a DynamicColor, resolved at draw time.
func DynStop(position float32, dc DynamicColor) ColorStop {
	return ColorStop{Dynamic: dc, Position: position}
}

// coordSource selects which coordinate system to use for gradient computation.
type coordSource int

const (
	coordScreen coordSource = iota
	coordPane
	coordLocal
)

// oklchStop holds a pre-converted OKLCH color stop for interpolation.
type oklchStop struct {
	L, C, H  float64
	position float64
	rgb      tcell.Color // original RGB, avoids OKLCH round-trip at stop boundaries
}

// GradientBuilder configures and builds gradient DynamicColors.
type GradientBuilder struct {
	linear   bool
	angleDeg float32
	cx, cy   float32
	stops    []ColorStop
	source   coordSource
}

// Linear creates a linear gradient builder with the given angle and color stops.
// Angle: 0 = left-to-right, 90 = top-to-bottom, 45 = diagonal.
func Linear(angleDeg float32, stops ...ColorStop) GradientBuilder {
	return GradientBuilder{
		linear:   true,
		angleDeg: angleDeg,
		stops:    stops,
	}
}

// Radial creates a radial gradient builder centered at (cx, cy) in normalized
// coordinates (0-1). Distance from center determines gradient position.
func Radial(cx, cy float32, stops ...ColorStop) GradientBuilder {
	return GradientBuilder{
		linear: false,
		cx:     cx,
		cy:     cy,
		stops:  stops,
	}
}

// WithLocal sets the gradient to use widget-local coordinates.
func (gb GradientBuilder) WithLocal() GradientBuilder {
	gb.source = coordLocal
	return gb
}

// WithPane sets the gradient to use pane coordinates.
func (gb GradientBuilder) WithPane() GradientBuilder {
	gb.source = coordPane
	return gb
}

// withSource sets the coordinate source directly (used by FromDesc reconstruction).
func (gb GradientBuilder) withSource(s coordSource) GradientBuilder {
	gb.source = s
	return gb
}

// buildDesc converts a GradientBuilder into a DynamicColorDesc.
func (gb GradientBuilder) buildDesc() DynamicColorDesc {
	stops := make([]GradientStopDesc, len(gb.stops))
	for i, s := range gb.stops {
		if !s.Dynamic.IsZero() {
			stops[i] = GradientStopDesc{Position: s.Position, Color: s.Dynamic.Describe()}
		} else {
			r, g, b := s.Color.RGB()
			stops[i] = GradientStopDesc{
				Position: s.Position,
				Color:    DynamicColorDesc{Type: DescTypeSolid, Base: PackRGB(r, g, b)},
			}
		}
	}

	desc := DynamicColorDesc{Stops: stops, Easing: uint8(gb.source)}
	if gb.linear {
		desc.Type = DescTypeLinearGrad
		desc.Base = math.Float32bits(gb.angleDeg)
	} else {
		desc.Type = DescTypeRadialGrad
		desc.Speed = gb.cx
		desc.Target = math.Float32bits(gb.cy)
	}
	return desc
}

// Build produces a DynamicColor from the gradient configuration.
func (gb GradientBuilder) Build() DynamicColor {
	source := gb.source
	desc := gb.buildDesc()

	// Check if any stop is dynamic (needs per-frame resolution).
	hasDyn := false
	for _, s := range gb.stops {
		if !s.Dynamic.IsZero() && !s.Dynamic.IsStatic() {
			hasDyn = true
			break
		}
	}

	if len(gb.stops) == 0 {
		return Solid(tcell.NewRGBColor(0, 0, 0))
	}
	if len(gb.stops) == 1 {
		if !gb.stops[0].Dynamic.IsZero() {
			dc := gb.stops[0].Dynamic
			dc.desc = desc
			return dc
		}
		s := Solid(gb.stops[0].Color)
		s.desc = desc
		return s
	}

	// Capture stops for the closure.
	capturedStops := gb.stops

	var fn ColorFunc
	if gb.linear {
		angleDeg := gb.angleDeg
		if hasDyn {
			fn = func(ctx ColorContext) tcell.Color {
				resolved := resolveStops(capturedStops, ctx)
				nx, ny := normalizedCoords(ctx, source)
				rad := float64(angleDeg) * math.Pi / 180.0
				t := nx*math.Cos(rad) + ny*math.Sin(rad)
				t = clampFloat(t, 0, 1)
				return interpolateStops(resolved, t)
			}
		} else {
			static := prepareStops(capturedStops)
			fn = func(ctx ColorContext) tcell.Color {
				nx, ny := normalizedCoords(ctx, source)
				rad := float64(angleDeg) * math.Pi / 180.0
				t := nx*math.Cos(rad) + ny*math.Sin(rad)
				t = clampFloat(t, 0, 1)
				return interpolateStops(static, t)
			}
		}
	} else {
		cx, cy := gb.cx, gb.cy
		if hasDyn {
			fn = func(ctx ColorContext) tcell.Color {
				resolved := resolveStops(capturedStops, ctx)
				nx, ny := normalizedCoords(ctx, source)
				dx := nx - float64(cx)
				dy := ny - float64(cy)
				t := math.Sqrt(dx*dx+dy*dy) * 2
				t = clampFloat(t, 0, 1)
				return interpolateStops(resolved, t)
			}
		} else {
			static := prepareStops(capturedStops)
			fn = func(ctx ColorContext) tcell.Color {
				nx, ny := normalizedCoords(ctx, source)
				dx := nx - float64(cx)
				dy := ny - float64(cy)
				t := math.Sqrt(dx*dx+dy*dy) * 2
				t = clampFloat(t, 0, 1)
				return interpolateStops(static, t)
			}
		}
	}

	return DynamicColor{
		fn:       fn,
		animated: hasDyn,
		desc:     desc,
	}
}

// hasDynamicStops reports whether any stop uses a DynamicColor.
func hasDynamicStops(stops []ColorStop) bool {
	for _, s := range stops {
		if !s.Dynamic.IsZero() {
			return true
		}
	}
	return false
}

// resolveStops resolves DynamicColor stops with the given context, then converts to oklchStops.
func resolveStops(stops []ColorStop, ctx ColorContext) []oklchStop {
	sorted := make([]ColorStop, len(stops))
	copy(sorted, stops)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	out := make([]oklchStop, len(sorted))
	for i, s := range sorted {
		c := s.Color
		if !s.Dynamic.IsZero() {
			c = s.Dynamic.Resolve(ctx)
		}
		l, ch, h := TcellToOKLCH(c)
		out[i] = oklchStop{L: l, C: ch, H: h, position: float64(s.Position), rgb: c}
	}
	return out
}

// prepareStops converts static ColorStops to oklchStops, sorted by position.
func prepareStops(stops []ColorStop) []oklchStop {
	return resolveStops(stops, ColorContext{})
}

// normalizedCoords returns (nx, ny) in [0,1] based on the coordinate source.
func normalizedCoords(ctx ColorContext, source coordSource) (float64, float64) {
	var x, y, w, h float64
	switch source {
	case coordLocal:
		x, y = float64(ctx.X), float64(ctx.Y)
		w, h = float64(ctx.W), float64(ctx.H)
	case coordPane:
		x, y = float64(ctx.PX), float64(ctx.PY)
		w, h = float64(ctx.PW), float64(ctx.PH)
	default:
		x, y = float64(ctx.SX), float64(ctx.SY)
		w, h = float64(ctx.SW), float64(ctx.SH)
	}
	nx := x / math.Max(w-1, 1)
	ny := y / math.Max(h-1, 1)
	return nx, ny
}

// interpolateStops finds the two surrounding stops and interpolates between them.
// Returns the original RGB at stop boundaries to avoid OKLCH round-trip color shift.
func interpolateStops(stops []oklchStop, t float64) tcell.Color {
	if t <= stops[0].position {
		return stops[0].rgb
	}
	last := stops[len(stops)-1]
	if t >= last.position {
		return last.rgb
	}

	for i := 1; i < len(stops); i++ {
		if t <= stops[i].position {
			a := stops[i-1]
			b := stops[i]
			span := b.position - a.position
			if span <= 0 {
				return a.rgb
			}
			f := (t - a.position) / span
			// At exact stop boundaries, return original RGB.
			if f <= 0 {
				return a.rgb
			}
			if f >= 1 {
				return b.rgb
			}
			L := a.L + (b.L-a.L)*f
			C := a.C + (b.C-a.C)*f
			H := lerpHue(a.H, b.H, f)
			return OKLCHToTcell(L, C, H)
		}
	}

	return last.rgb
}

// lerpHue interpolates between two hue angles using the shortest arc around 360.
func lerpHue(h1, h2, t float64) float64 {
	diff := h2 - h1
	if diff > 180 {
		diff -= 360
	} else if diff < -180 {
		diff += 360
	}
	h := h1 + diff*t
	if h < 0 {
		h += 360
	} else if h >= 360 {
		h -= 360
	}
	return h
}

// clampFloat restricts a float64 value to the range [min, max].
func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
