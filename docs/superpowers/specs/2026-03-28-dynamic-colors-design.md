# Dynamic Colors Design

**Date:** 2026-03-28
**Status:** Approved
**Scope:** `texelui/color/`, `texelui/animation/`, `texelui/core/painter.go`, all widget Draw methods

## Summary

Replace static `tcell.Color` in widget styling with `DynamicColor` — a type that can be either a plain color or a function of position and time. This enables spatial gradients (linear, radial), animated colors (pulsing, cycling), and arbitrary color effects, all interpolated in OKLCH color space for perceptual smoothness.

## Core Types

### ColorContext

The input to every color function. Carries three coordinate systems plus time:

```go
type ColorContext struct {
    // Widget-local coordinates (origin = widget top-left)
    X, Y   int
    W, H   int   // widget dimensions for normalization

    // Pane coordinates (origin = containing pane top-left)
    PX, PY int
    PW, PH int

    // Screen-absolute coordinates (origin = terminal top-left)
    SX, SY int
    SW, SH int

    // Animation time (0.0–1.0 eased progress, or continuous)
    T      float32
}
```

**Coordinate use cases:**
- Local (X, Y): per-widget glow, button highlight
- Pane (PX, PY): shared gradient across all widgets in a panel
- Screen (SX, SY): screen-wide wallpaper effect
- Time (T): color cycling, breathing, transitions

### ColorFunc

The universal color function signature:

```go
type ColorFunc func(ctx ColorContext) tcell.Color
```

### DynamicColor

A color that can be static or dynamic. Constructed once, assigned to any number of widgets. Multiple widgets sharing the same DynamicColor instance sample from the same color field — their colors align spatially and temporally.

```go
type DynamicColor struct {
    static   tcell.Color  // used when fn is nil
    fn       ColorFunc    // used when non-nil
    animated bool         // true if color depends on time (T)
}

func (dc DynamicColor) Resolve(ctx ColorContext) tcell.Color {
    if dc.fn != nil {
        return dc.fn(ctx)
    }
    return dc.static
}

func (dc DynamicColor) IsStatic() bool   { return dc.fn == nil }
func (dc DynamicColor) IsAnimated() bool { return dc.animated }
```

**Constructors:**

```go
func Solid(c tcell.Color) DynamicColor                          // static color
func Func(fn ColorFunc) DynamicColor                            // custom spatial function
func AnimatedFunc(fn ColorFunc) DynamicColor                    // custom function using T
func Linear(angle float32, stops ...ColorStop) DynamicColor     // linear gradient
func Radial(cx, cy float32, stops ...ColorStop) DynamicColor    // radial gradient
```

The `animated` flag is set explicitly by constructors. `Solid`, `Func`, `Linear`, `Radial` set `animated = false`. `AnimatedFunc` and any time-dependent builder (e.g., `Cycle`, `Breathe`) set `animated = true`. The Painter checks `IsAnimated()` per cell to determine if the frame needs continued redraws.

### DynamicStyle

Replaces `tcell.Style` in widget APIs where dynamic colors are needed:

```go
type DynamicStyle struct {
    FG    DynamicColor
    BG    DynamicColor
    Attrs tcell.AttrMask  // Bold, Dim, Italic, Underline, etc.
    URL   string          // OSC 8 hyperlink (from tcell.Style)
}
```

**Conversion from `tcell.Style`:**

```go
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

`StyleFrom` extracts all fields from `tcell.Style` including attributes and URL, ensuring no capability is lost.

### ColorStop

A point in a gradient:

```go
type ColorStop struct {
    Color    tcell.Color
    Position float32       // 0.0 to 1.0
}

func Stop(position float32, color tcell.Color) ColorStop
```

## Gradient Builder

### Linear Gradients

Defined by an angle (degrees) and color stops:
- 0° = left to right
- 90° = top to bottom
- 45° = diagonal top-left to bottom-right

The gradient projects each position onto the gradient axis, normalizes to [0, 1], then interpolates between stops.

**Coordinate system selection:** Gradient builders default to screen coordinates (SX/SY/SW/SH) so shared gradients align across widgets. Modifiers select different coordinate sources:

```go
rainbow := color.Linear(0,
    color.Stop(0.0, red),
    color.Stop(0.5, purple),
    color.Stop(1.0, blue),
)                                    // screen coordinates (default)

buttonGlow := color.Linear(90,
    color.Stop(0.0, bright),
    color.Stop(1.0, dark),
).WithLocal()                        // widget-local coordinates

paneBg := color.Linear(45,
    color.Stop(0.0, warm),
    color.Stop(1.0, cool),
).WithPane()                         // pane coordinates
```

### Radial Gradients

Defined by a center point (normalized 0–1) and color stops:

```go
spotlight := color.Radial(0.5, 0.5,
    color.Stop(0.0, white),
    color.Stop(1.0, black),
).WithLocal()
```

### OKLCH Interpolation

All gradient interpolation uses **float64 precision** internally for color science accuracy (float32 causes visible banding). The `float32` type is used only at the API boundary (stop positions, T value).

The existing `texelui/color/oklch.go` already provides `RGBToOKLCH` and `OKLCHToRGB` (float64). The existing `color/spaces.go` provides `TcellToOKLCH` and `OKLCHToTcell`. No new OKLCH math needs to be written — gradient.go calls these directly.

Interpolation steps:
1. Convert each stop's `tcell.Color` → OKLCH (via existing helpers)
2. Interpolate L, C, H independently between stops (float64)
3. Hue interpolation takes the shortest arc (handles 360° wrap)
4. Convert result OKLCH → `tcell.Color` (via existing helpers)

## Rendering Pipeline

### Painter Changes

The Painter carries context for dynamic color resolution:

```go
type Painter struct {
    buf        [][]Cell
    clip       Rect
    gp         GraphicsProvider

    // Dynamic color context
    widgetRect Rect     // actual widget bounds (not clip)
    paneRect   Rect     // containing pane bounds
    screenW    int      // terminal width
    screenH    int      // terminal height
    time       float32  // current animation time
}
```

**Widget-local coordinate derivation:** Local coordinates use `widgetRect` (the widget's actual bounds), NOT the clip rect. The clip rect can differ from widget bounds inside `ScrollPane` or `Border` containers. `widgetRect` is set by the UIManager before each `Draw` call.

**`WithClip` propagation:** `WithClip` must propagate ALL context fields (`widgetRect`, `paneRect`, `screenW`, `screenH`, `time`) to the new Painter. Currently it only copies `buf`, `clip`, and `gp`.

**New method:**

```go
func (p *Painter) SetDynamicCell(x, y int, ch rune, ds DynamicStyle) {
    // clipping checks...

    ctx := ColorContext{
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
    if ds.Attrs != 0 {
        style = style.Attributes(ds.Attrs)
    }
    if ds.URL != "" {
        style = style.Url(ds.URL)
    }

    p.buf[y][x] = Cell{Ch: ch, Style: style}
}
```

### Cell Unchanged

`Cell` continues to store resolved `tcell.Style`. Dynamic resolution happens at write time in the Painter. This means:
- No changes to the client/server protocol
- No changes to the rendering output path
- Static DynamicColors resolve with no allocation overhead (fast path)

### Widget Migration

Widgets that want dynamic colors use `SetDynamicCell` instead of `SetCell`. The existing `SetCell` continues to work. Migration is incremental — one widget at a time.

## Animation System

### Timeline (moved from texelation)

The existing `Timeline` + `EasingFunc` from `texelation/internal/effects/timeline.go` moves to `texelui/animation/`. It has zero dependencies beyond `sync` and `time`. It provides:

- Per-key animation state with configurable easing
- `AnimateTo(key, target, duration, now)` → current eased value
- Common easing functions: linear, smoothstep, quadratic, cubic

### Time in ColorContext

The `T` field can represent:
- **Eased progress** (0→1): from a Timeline animation, for transitions
- **Continuous cycle**: `time.Now` modulo a period, for looping effects

The Painter receives the current time value from the UIManager render loop.

### Redraw Scheduling

The Painter tracks whether any `SetDynamicCell` call involved an animated DynamicColor (via `IsAnimated()`). If any animated color was resolved during a frame, the UIManager continues scheduling redraws. Static gradients (position-only, no time) don't trigger redraws.

## Package Layout

| Package | Files | Responsibility |
|---------|-------|---------------|
| `texelui/color` | `dynamic.go`, `gradient.go` | DynamicColor, ColorFunc, ColorContext, DynamicStyle, gradient builders |
| `texelui/color` | `oklch.go`, `spaces.go` | Already exists — OKLCH conversion |
| `texelui/animation` | `timeline.go`, `easing.go` | Timeline, EasingFunc (moved from texelation) |
| `texelui/core` | `painter.go` (modified) | SetDynamicCell, context propagation, WithClip updates |

**Dependency constraint:** `texelui/color` must NOT import `texelui/core`. All types in `ColorContext` are plain `int`/`float32` — no dependency on `core.Rect`.

## Backward Compatibility

- `Cell` type unchanged
- `Painter.SetCell` continues to work (not removed)
- `tcell.Style` accepted everywhere via `StyleFrom()`
- Widgets migrate incrementally — no big-bang rewrite
- `Solid(color)` wraps existing `tcell.Color` with zero overhead

## Testing

- OKLCH round-trip: RGB → OKLCH → RGB preserves colors within tolerance
- Gradient interpolation: verify color stops at 0.0, 0.5, 1.0 produce correct colors
- Linear gradient angles: 0° horizontal, 90° vertical, 45° diagonal
- Radial gradient: center produces stop 0 color, edge produces stop 1 color
- DynamicColor.Resolve: static returns color directly, function receives correct context
- Coordinate systems: verify local/pane/screen coordinates differ correctly in ColorContext
- WithClip propagation: verify dynamic context survives through WithClip calls
- Animation: Timeline easing produces correct T values over time
- IsAnimated: verify constructors set the flag correctly
- Performance: static DynamicColor resolves with no allocation overhead
- StyleFrom: verify all fields extracted (FG, BG, Attrs, URL)
