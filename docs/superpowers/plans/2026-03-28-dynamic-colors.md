# Dynamic Colors Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add DynamicColor type to texelui that enables spatial gradients, animated colors, and arbitrary color functions, all interpolated in OKLCH.

**Architecture:** Bottom-up: animation package (Timeline) → color types (DynamicColor/ColorFunc/DynamicStyle) → gradient builders (Linear/Radial with OKLCH) → Painter integration (SetDynamicCell with context) → demo proof.

**Tech Stack:** Go, tcell, OKLCH color math (existing in `color/oklch.go` and `color/spaces.go`)

**Spec:** `docs/superpowers/specs/2026-03-28-dynamic-colors-design.md`

---

### Task 1: Move Timeline and EasingFunc to texelui/animation

**Files:**
- Create: `animation/timeline.go`
- Create: `animation/easing.go`
- Create: `animation/timeline_test.go`

This is a direct port from `texelation/internal/effects/timeline.go` (309 lines). Split into two files: `easing.go` (EasingFunc type + all easing functions) and `timeline.go` (Timeline struct + AnimateOptions). No dependencies beyond `sync` and `time`.

- [ ] **Step 1: Create `animation/easing.go`**

Port the `EasingFunc` type and all easing function variables from the texelation source. Change package from `effects` to `animation`.

```go
package animation

// EasingFunc maps progress [0,1] to eased value [0,1]
type EasingFunc func(progress float32) float32

var (
    EaseLinear       EasingFunc = func(t float32) float32 { return t }
    EaseSmoothstep   EasingFunc = func(t float32) float32 { return t * t * (3.0 - 2.0*t) }
    EaseSmootherstep EasingFunc = func(t float32) float32 { return t * t * t * (t*(t*6.0-15.0) + 10.0) }
    EaseInQuad       EasingFunc = func(t float32) float32 { return t * t }
    EaseOutQuad      EasingFunc = func(t float32) float32 { return t * (2.0 - t) }
    EaseInOutQuad    EasingFunc = func(t float32) float32 { /* ... */ }
    EaseInCubic      EasingFunc = func(t float32) float32 { return t * t * t }
    EaseOutCubic     EasingFunc = func(t float32) float32 { /* ... */ }
    EaseInOutCubic   EasingFunc = func(t float32) float32 { /* ... */ }
)
```

Copy the full implementations from `/home/marc/projects/texel/texelation/internal/effects/timeline.go` lines 16-72.

- [ ] **Step 2: Create `animation/timeline.go`**

Port `AnimateOptions`, `DefaultAnimateOptions`, `keyState`, `Timeline`, and all Timeline methods. Change package from `effects` to `animation`. Copy from texelation source lines 74-309.

- [ ] **Step 3: Write tests `animation/timeline_test.go`**

```go
func TestEasings(t *testing.T) {
    // All easings should return 0 at 0 and 1 at 1
    easings := []animation.EasingFunc{
        animation.EaseLinear, animation.EaseSmoothstep,
        animation.EaseInQuad, animation.EaseOutQuad,
    }
    for _, e := range easings {
        if e(0) != 0 { t.Error("easing(0) should be 0") }
        if e(1) != 1 { t.Error("easing(1) should be 1") }
    }
}

func TestTimeline_AnimateTo(t *testing.T) {
    tl := animation.NewTimeline(0.0)
    now := time.Now()
    // Start animation from 0 to 1 over 100ms
    v := tl.AnimateTo("key", 1.0, 100*time.Millisecond, now)
    if v != 0.0 { t.Errorf("initial value should be 0, got %f", v) }
    // At half duration
    v = tl.Get("key", now.Add(50*time.Millisecond))
    if v < 0.1 || v > 0.9 { t.Errorf("mid value should be ~0.5, got %f", v) }
    // At end
    v = tl.Get("key", now.Add(200*time.Millisecond))
    if v != 1.0 { t.Errorf("final value should be 1.0, got %f", v) }
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./animation/ -v`

- [ ] **Step 5: Commit**

```bash
git add animation/
git commit -m "Add animation package with Timeline and easing functions

Ported from texelation/internal/effects/timeline.go to make animation
primitives available to texelui widgets for dynamic colors."
```

---

### Task 2: Core DynamicColor types

**Files:**
- Create: `color/dynamic.go`
- Create: `color/dynamic_test.go`

No dependencies on `core` package. Only imports `tcell`.

- [ ] **Step 1: Create `color/dynamic.go` with core types**

```go
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
    T      float32 // Animation time (0.0-1.0 or continuous)
}

// ColorFunc computes a color from spatial and temporal context.
type ColorFunc func(ctx ColorContext) tcell.Color

// DynamicColor is a color that can be static or a function of position/time.
// Multiple widgets sharing the same instance sample the same color field.
type DynamicColor struct {
    static   tcell.Color
    fn       ColorFunc
    animated bool
}

// Solid creates a static DynamicColor from a plain tcell.Color.
func Solid(c tcell.Color) DynamicColor {
    return DynamicColor{static: c}
}

// Func creates a spatial DynamicColor (no time dependency).
func Func(fn ColorFunc) DynamicColor {
    return DynamicColor{fn: fn}
}

// AnimatedFunc creates a time-dependent DynamicColor.
func AnimatedFunc(fn ColorFunc) DynamicColor {
    return DynamicColor{fn: fn, animated: true}
}

// Resolve returns the concrete tcell.Color for the given context.
func (dc DynamicColor) Resolve(ctx ColorContext) tcell.Color {
    if dc.fn != nil {
        return dc.fn(ctx)
    }
    return dc.static
}

// IsStatic returns true if this color has no function (always the same color).
func (dc DynamicColor) IsStatic() bool { return dc.fn == nil }

// IsAnimated returns true if this color depends on time.
func (dc DynamicColor) IsAnimated() bool { return dc.animated }

// DynamicStyle combines dynamic FG/BG colors with attributes.
type DynamicStyle struct {
    FG    DynamicColor
    BG    DynamicColor
    Attrs tcell.AttrMask
    URL   string
}

// StyleFrom converts a tcell.Style to a DynamicStyle with static colors.
func StyleFrom(s tcell.Style) DynamicStyle {
    fg, bg, attrs := s.Decompose()
    return DynamicStyle{
        FG:    Solid(fg),
        BG:    Solid(bg),
        Attrs: attrs,
        URL:   s.Url(),
    }
}
```

- [ ] **Step 2: Write tests `color/dynamic_test.go`**

```go
func TestSolid_Resolve(t *testing.T) {
    c := Solid(tcell.ColorRed)
    ctx := ColorContext{}
    if c.Resolve(ctx) != tcell.ColorRed {
        t.Error("Solid should return its color")
    }
    if !c.IsStatic() { t.Error("Solid should be static") }
    if c.IsAnimated() { t.Error("Solid should not be animated") }
}

func TestFunc_Resolve(t *testing.T) {
    c := Func(func(ctx ColorContext) tcell.Color {
        if ctx.X > 5 { return tcell.ColorBlue }
        return tcell.ColorRed
    })
    if c.Resolve(ColorContext{X: 0}) != tcell.ColorRed { t.Error("x=0 should be red") }
    if c.Resolve(ColorContext{X: 10}) != tcell.ColorBlue { t.Error("x=10 should be blue") }
    if c.IsStatic() { t.Error("Func should not be static") }
    if c.IsAnimated() { t.Error("Func should not be animated") }
}

func TestAnimatedFunc(t *testing.T) {
    c := AnimatedFunc(func(ctx ColorContext) tcell.Color { return tcell.ColorGreen })
    if !c.IsAnimated() { t.Error("AnimatedFunc should be animated") }
}

func TestStyleFrom(t *testing.T) {
    s := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlue).Bold(true)
    ds := StyleFrom(s)
    ctx := ColorContext{}
    if ds.FG.Resolve(ctx) != tcell.ColorRed { t.Error("FG mismatch") }
    if ds.BG.Resolve(ctx) != tcell.ColorBlue { t.Error("BG mismatch") }
    if ds.Attrs&tcell.AttrBold == 0 { t.Error("Bold attr missing") }
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./color/ -v -run TestSolid\|TestFunc\|TestAnimated\|TestStyleFrom`

- [ ] **Step 4: Commit**

```bash
git add color/dynamic.go color/dynamic_test.go
git commit -m "Add DynamicColor, ColorFunc, ColorContext, DynamicStyle core types"
```

---

### Task 3: Gradient builders with OKLCH interpolation

**Files:**
- Create: `color/gradient.go`
- Create: `color/gradient_test.go`

Uses existing `color/oklch.go` and `color/spaces.go` for OKLCH math. All interpolation in float64.

- [ ] **Step 1: Create `color/gradient.go`**

```go
package color

import (
    "math"
    "github.com/gdamore/tcell/v2"
)

// ColorStop defines a color at a position in a gradient.
type ColorStop struct {
    Color    tcell.Color
    Position float32 // 0.0 to 1.0
}

// Stop creates a ColorStop.
func Stop(position float32, c tcell.Color) ColorStop {
    return ColorStop{Color: c, Position: position}
}

// coordSource selects which coordinates the gradient reads.
type coordSource int
const (
    coordScreen coordSource = iota
    coordPane
    coordLocal
)

// gradientDef holds gradient parameters for deferred ColorFunc construction.
type gradientDef struct {
    stops  []ColorStop
    source coordSource
}

// WithLocal returns a DynamicColor using widget-local coordinates.
func (g gradientDef) WithLocal() DynamicColor {
    g.source = coordLocal
    return g.build()
}

// WithPane returns a DynamicColor using pane coordinates.
func (g gradientDef) WithPane() DynamicColor {
    g.source = coordPane
    return g.build()
}
```

The `Linear` function:

```go
// Linear creates a linear gradient at the given angle (degrees).
// 0° = left to right, 90° = top to bottom.
// Default coordinate source: screen.
func Linear(angleDeg float32, stops ...ColorStop) gradientDef {
    // ... store angle and stops, return gradientDef
}

func (g gradientDef) build() DynamicColor {
    // Pre-convert all stops to OKLCH (float64)
    // Return DynamicColor{fn: func that projects coords, interpolates in OKLCH}
}
```

The interpolation function:

```go
// interpolateOKLCH interpolates between OKLCH stops at position t (0-1).
func interpolateOKLCH(stops []oklchStop, t float64) tcell.Color {
    // Find surrounding stops, lerp L/C/H, shortest hue arc, convert back
}

// lerpHue interpolates hue taking the shortest arc around 360°.
func lerpHue(h1, h2, t float64) float64 {
    diff := h2 - h1
    if diff > 180 { diff -= 360 }
    if diff < -180 { diff += 360 }
    h := h1 + diff*t
    if h < 0 { h += 360 }
    if h >= 360 { h -= 360 }
    return h
}
```

The `Radial` function follows the same pattern but uses distance from center.

- [ ] **Step 2: Write tests `color/gradient_test.go`**

```go
func TestLinearGradient_Horizontal(t *testing.T) {
    g := Linear(0, Stop(0, tcell.ColorRed), Stop(1, tcell.ColorBlue)).WithLocal()
    // At left edge (x=0, w=10): should be close to red
    left := g.Resolve(ColorContext{X: 0, W: 10, H: 1})
    // At right edge (x=9, w=10): should be close to blue
    right := g.Resolve(ColorContext{X: 9, W: 10, H: 1})
    // They should be different colors
    if left == right { t.Error("gradient endpoints should differ") }
}

func TestLinearGradient_Vertical(t *testing.T) {
    g := Linear(90, Stop(0, tcell.ColorRed), Stop(1, tcell.ColorBlue)).WithLocal()
    top := g.Resolve(ColorContext{Y: 0, W: 1, H: 10})
    bottom := g.Resolve(ColorContext{Y: 9, W: 1, H: 10})
    if top == bottom { t.Error("vertical gradient endpoints should differ") }
}

func TestLinearGradient_StopAtExactPosition(t *testing.T) {
    red := tcell.NewRGBColor(255, 0, 0)
    g := Linear(0, Stop(0, red), Stop(1, tcell.ColorBlue)).WithLocal()
    // At position 0.0, should be exactly red
    c := g.Resolve(ColorContext{X: 0, W: 100, H: 1})
    if c != red { t.Errorf("stop at 0.0 should be red, got %v", c) }
}

func TestRadialGradient(t *testing.T) {
    g := Radial(0.5, 0.5, Stop(0, tcell.ColorWhite), Stop(1, tcell.ColorBlack)).WithLocal()
    center := g.Resolve(ColorContext{X: 5, Y: 5, W: 10, H: 10})
    corner := g.Resolve(ColorContext{X: 0, Y: 0, W: 10, H: 10})
    if center == corner { t.Error("center and corner should differ") }
}

func TestLerpHue_ShortestArc(t *testing.T) {
    // From 350° to 10° should go through 0°, not through 180°
    h := lerpHue(350, 10, 0.5)
    if h < 355 && h > 5 { t.Errorf("hue should be near 0/360, got %f", h) }
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./color/ -v -run TestLinear\|TestRadial\|TestLerp`

- [ ] **Step 4: Commit**

```bash
git add color/gradient.go color/gradient_test.go
git commit -m "Add Linear and Radial gradient builders with OKLCH interpolation"
```

---

### Task 4: Painter integration — SetDynamicCell and context propagation

**Files:**
- Modify: `core/painter.go`
- Create: `core/painter_dynamic_test.go`

The Painter gets new fields for dynamic color context and a new `SetDynamicCell` method. `WithClip` propagates all new fields.

**Important:** `core/painter.go` imports `tcell` only. The new `SetDynamicCell` needs `color.DynamicStyle` and `color.ColorContext` from the `color` package. This creates an import: `core` → `color`. Verify this doesn't create a cycle (`color` must NOT import `core`).

- [ ] **Step 1: Add context fields to Painter**

Add to the `Painter` struct:

```go
type Painter struct {
    buf        [][]Cell
    clip       Rect
    gp         GraphicsProvider
    // Dynamic color context
    widgetRect Rect
    paneRect   Rect
    screenW    int
    screenH    int
    time       float32
    hasAnim    bool // set true if any animated color resolved
}
```

Add setters:

```go
func (p *Painter) SetWidgetRect(r Rect)  { p.widgetRect = r }
func (p *Painter) SetPaneRect(r Rect)    { p.paneRect = r }
func (p *Painter) SetScreenSize(w, h int) { p.screenW = w; p.screenH = h }
func (p *Painter) SetTime(t float32)     { p.time = t }
func (p *Painter) HasAnimations() bool   { return p.hasAnim }
```

- [ ] **Step 2: Update WithClip to propagate context**

```go
func (p *Painter) WithClip(rect Rect) *Painter {
    // ... existing intersection logic ...
    return &Painter{
        buf:        p.buf,
        clip:       Rect{X: left, Y: top, W: right - left, H: bottom - top},
        gp:         p.gp,
        widgetRect: p.widgetRect,
        paneRect:   p.paneRect,
        screenW:    p.screenW,
        screenH:    p.screenH,
        time:       p.time,
    }
}
```

Also update the empty-clip return path to propagate fields.

- [ ] **Step 3: Add SetDynamicCell**

```go
import "github.com/framegrace/texelui/color"

func (p *Painter) SetDynamicCell(x, y int, ch rune, ds color.DynamicStyle) {
    if p.buf == nil {
        return
    }
    if x < p.clip.X || y < p.clip.Y || x >= p.clip.X+p.clip.W || y >= p.clip.Y+p.clip.H {
        return
    }
    if y < 0 || y >= len(p.buf) || x < 0 || x >= len(p.buf[y]) {
        return
    }

    // Track animation
    if ds.FG.IsAnimated() || ds.BG.IsAnimated() {
        p.hasAnim = true
    }

    // Fast path: both static
    if ds.FG.IsStatic() && ds.BG.IsStatic() {
        style := tcell.StyleDefault.Foreground(ds.FG.Resolve(color.ColorContext{})).
            Background(ds.BG.Resolve(color.ColorContext{}))
        if ds.Attrs != 0 { style = style.Attributes(ds.Attrs) }
        if ds.URL != "" { style = style.Url(ds.URL) }
        p.buf[y][x] = Cell{Ch: ch, Style: style}
        return
    }

    ctx := color.ColorContext{
        X:  x - p.widgetRect.X,  Y:  y - p.widgetRect.Y,
        W:  p.widgetRect.W,      H:  p.widgetRect.H,
        PX: x - p.paneRect.X,   PY: y - p.paneRect.Y,
        PW: p.paneRect.W,       PH: p.paneRect.H,
        SX: x,                  SY: y,
        SW: p.screenW,          SH: p.screenH,
        T:  p.time,
    }

    fg := ds.FG.Resolve(ctx)
    bg := ds.BG.Resolve(ctx)
    style := tcell.StyleDefault.Foreground(fg).Background(bg)
    if ds.Attrs != 0 { style = style.Attributes(ds.Attrs) }
    if ds.URL != "" { style = style.Url(ds.URL) }
    p.buf[y][x] = Cell{Ch: ch, Style: style}
}
```

Also add convenience methods:

```go
func (p *Painter) FillDynamic(rect Rect, ch rune, ds color.DynamicStyle) {
    for yy := rect.Y; yy < rect.Y+rect.H; yy++ {
        for xx := rect.X; xx < rect.X+rect.W; xx++ {
            p.SetDynamicCell(xx, yy, ch, ds)
        }
    }
}

func (p *Painter) DrawDynamicText(x, y int, s string, ds color.DynamicStyle) {
    xx := x
    for _, r := range s {
        p.SetDynamicCell(xx, y, r, ds)
        xx++
    }
}
```

- [ ] **Step 4: Write tests `core/painter_dynamic_test.go`**

```go
func TestSetDynamicCell_Static(t *testing.T) {
    buf := makeCellBuf(10, 1)
    p := NewPainter(buf, Rect{X: 0, Y: 0, W: 10, H: 1})
    ds := color.StyleFrom(tcell.StyleDefault.Foreground(tcell.ColorRed))
    p.SetDynamicCell(0, 0, 'A', ds)
    if buf[0][0].Ch != 'A' { t.Error("char mismatch") }
}

func TestSetDynamicCell_Gradient(t *testing.T) {
    buf := makeCellBuf(10, 1)
    p := NewPainter(buf, Rect{X: 0, Y: 0, W: 10, H: 1})
    p.SetWidgetRect(Rect{X: 0, Y: 0, W: 10, H: 1})
    g := color.Linear(0, color.Stop(0, tcell.ColorRed), color.Stop(1, tcell.ColorBlue)).WithLocal()
    ds := color.DynamicStyle{FG: g, BG: color.Solid(tcell.ColorBlack)}
    p.SetDynamicCell(0, 0, 'A', ds)
    p.SetDynamicCell(9, 0, 'B', ds)
    // Colors at x=0 and x=9 should differ
    fg0, _, _ := buf[0][0].Style.Decompose()
    fg9, _, _ := buf[0][9].Style.Decompose()
    if fg0 == fg9 { t.Error("gradient should produce different colors at different positions") }
}

func TestWithClip_PropagatesContext(t *testing.T) {
    buf := makeCellBuf(20, 20)
    p := NewPainter(buf, Rect{X: 0, Y: 0, W: 20, H: 20})
    p.SetPaneRect(Rect{X: 5, Y: 5, W: 10, H: 10})
    p.SetScreenSize(80, 24)
    p.SetTime(0.5)
    clipped := p.WithClip(Rect{X: 2, Y: 2, W: 10, H: 10})
    if clipped.paneRect.X != 5 { t.Error("paneRect not propagated through WithClip") }
    if clipped.screenW != 80 { t.Error("screenW not propagated") }
    if clipped.time != 0.5 { t.Error("time not propagated") }
}

func TestHasAnimations(t *testing.T) {
    buf := makeCellBuf(10, 1)
    p := NewPainter(buf, Rect{X: 0, Y: 0, W: 10, H: 1})
    ds := color.DynamicStyle{
        FG: color.AnimatedFunc(func(ctx color.ColorContext) tcell.Color { return tcell.ColorRed }),
        BG: color.Solid(tcell.ColorBlack),
    }
    if p.HasAnimations() { t.Error("should not have animations before draw") }
    p.SetDynamicCell(0, 0, 'X', ds)
    if !p.HasAnimations() { t.Error("should have animations after drawing animated color") }
}

func makeCellBuf(w, h int) [][]Cell {
    buf := make([][]Cell, h)
    for i := range buf { buf[i] = make([]Cell, w) }
    return buf
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./core/ -v -run TestSetDynamicCell\|TestWithClip_Propagates\|TestHasAnimations`

- [ ] **Step 6: Commit**

```bash
git add core/painter.go core/painter_dynamic_test.go
git commit -m "Add SetDynamicCell to Painter with spatial/temporal context propagation"
```

---

### Task 5: Demo — gradient tab in texelui-demo

**Files:**
- Modify: `apps/texelui-demo/demo.go`

Add a "Gradients" tab to the demo showcasing dynamic colors. This proves the full pipeline works end-to-end.

- [ ] **Step 1: Create a gradient demo pane**

Add a new function `createGradientsTab()` to `demo.go` that creates a pane with:

1. A label "Linear Gradients" drawn with a horizontal rainbow gradient FG
2. A filled rectangle with a vertical gradient BG
3. A filled rectangle with a radial gradient BG
4. A label explaining the coordinate systems

```go
func createGradientsTab() core.Widget {
    pane := widgets.NewPane()

    // The pane's Draw method will use SetDynamicCell for gradient backgrounds.
    // For now, demonstrate by using a custom widget that draws gradients.
    // ... (implementation details)

    return pane
}
```

Since existing widgets use `SetCell` (not yet migrated to `SetDynamicCell`), create a simple `GradientBox` custom widget inside the demo that calls `painter.SetDynamicCell` directly:

```go
type GradientBox struct {
    core.BaseWidget
    Style color.DynamicStyle
}

func (g *GradientBox) Draw(p *core.Painter) {
    p.SetWidgetRect(core.Rect{X: g.Rect.X, Y: g.Rect.Y, W: g.Rect.W, H: g.Rect.H})
    for y := g.Rect.Y; y < g.Rect.Y+g.Rect.H; y++ {
        for x := g.Rect.X; x < g.Rect.X+g.Rect.W; x++ {
            p.SetDynamicCell(x, y, ' ', g.Style)
        }
    }
}
```

- [ ] **Step 2: Wire the Gradients tab into the demo**

Add to `New()` in demo.go:

```go
tabPanel.AddTab("Gradients", createGradientsTab())
```

- [ ] **Step 3: Build and visually verify**

Run: `make build && ./bin/texelui-demo`

Navigate to the Gradients tab. Verify:
- Horizontal gradient shows smooth color transition left to right
- Vertical gradient shows smooth transition top to bottom
- Radial gradient shows concentric color rings
- Colors interpolate smoothly (no banding, no muddy midpoints)

- [ ] **Step 4: Commit**

```bash
git add apps/texelui-demo/demo.go
git commit -m "Add Gradients demo tab showcasing dynamic colors"
```
