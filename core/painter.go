package core

import (
	"github.com/framegrace/texelui/color"
	"github.com/gdamore/tcell/v2"
)

// Painter writes into a [][]Cell target with clipping.
type Painter struct {
	buf  [][]Cell
	clip Rect
	gp   GraphicsProvider
	// Dynamic color context
	widgetRect Rect
	paneRect   Rect
	screenW    int
	screenH    int
	time       float32
	hasAnim    bool
}

func NewPainter(buf [][]Cell, clip Rect) *Painter {
	return &Painter{buf: buf, clip: clip}
}

// NewPainterWithGraphics creates a Painter with a GraphicsProvider.
// Widgets can query the provider via GraphicsProvider() to decide their
// rendering strategy (e.g., Kitty image protocol vs half-block art).
func NewPainterWithGraphics(buf [][]Cell, clip Rect, gp GraphicsProvider) *Painter {
	return &Painter{buf: buf, clip: clip, gp: gp}
}

// GraphicsProvider returns the graphics provider, or nil if none was set.
func (p *Painter) GraphicsProvider() GraphicsProvider {
	return p.gp
}

func (p *Painter) Size() (int, int) {
	if p.buf == nil {
		return 0, 0
	}
	h := len(p.buf)
	w := 0
	if h > 0 {
		w = len(p.buf[0])
	}
	return w, h
}

// GetCell returns the character and style at the given position, or (' ', default) if out of bounds.
func (p *Painter) GetCell(x, y int) (rune, tcell.Style) {
	if p.buf != nil && y >= 0 && y < len(p.buf) && x >= 0 && x < len(p.buf[y]) {
		c := p.buf[y][x]
		return c.Ch, c.Style
	}
	return ' ', tcell.StyleDefault
}

func (p *Painter) SetCell(x, y int, ch rune, style tcell.Style) {
	if p.buf == nil {
		return
	}
	if x < p.clip.X || y < p.clip.Y || x >= p.clip.X+p.clip.W || y >= p.clip.Y+p.clip.H {
		return
	}
	if y >= 0 && y < len(p.buf) && x >= 0 && len(p.buf) > 0 && x < len(p.buf[y]) {
		p.buf[y][x] = Cell{Ch: ch, Style: style}
	}
}

// SetCellKeepBG writes a character and FG color but preserves the existing cell's background.
// Used by transparent widgets to overlay text on a parent's gradient/background.
func (p *Painter) SetCellKeepBG(x, y int, ch rune, style tcell.Style) {
	if p.buf == nil {
		return
	}
	if x < p.clip.X || y < p.clip.Y || x >= p.clip.X+p.clip.W || y >= p.clip.Y+p.clip.H {
		return
	}
	if y >= 0 && y < len(p.buf) && x >= 0 && len(p.buf) > 0 && x < len(p.buf[y]) {
		// Preserve existing BG, apply new FG + char + attrs
		_, existingBG, _ := p.buf[y][x].Style.Decompose()
		newFG, _, newAttrs := style.Decompose()
		p.buf[y][x] = Cell{Ch: ch, Style: tcell.StyleDefault.Foreground(newFG).Background(existingBG).Attributes(newAttrs)}
	}
}

func (p *Painter) Fill(rect Rect, ch rune, style tcell.Style) {
	for yy := rect.Y; yy < rect.Y+rect.H; yy++ {
		for xx := rect.X; xx < rect.X+rect.W; xx++ {
			p.SetCell(xx, yy, ch, style)
		}
	}
}

func (p *Painter) DrawText(x, y int, s string, style tcell.Style) {
	xx := x
	for _, r := range s {
		p.SetCell(xx, y, r, style)
		xx++
	}
}

func (p *Painter) DrawBorder(rect Rect, style tcell.Style, charset [6]rune) {
	if rect.W <= 1 || rect.H <= 1 {
		return
	}
	x0, y0 := rect.X, rect.Y
	x1, y1 := rect.X+rect.W-1, rect.Y+rect.H-1
	h, v := charset[0], charset[1]
	tl, tr, bl, br := charset[2], charset[3], charset[4], charset[5]
	for x := x0 + 1; x < x1; x++ {
		p.SetCell(x, y0, h, style)
		p.SetCell(x, y1, h, style)
	}
	for y := y0 + 1; y < y1; y++ {
		p.SetCell(x0, y, v, style)
		p.SetCell(x1, y, v, style)
	}
	p.SetCell(x0, y0, tl, style)
	p.SetCell(x1, y0, tr, style)
	p.SetCell(x0, y1, bl, style)
	p.SetCell(x1, y1, br, style)
}

// WithClip returns a new Painter that clips to the intersection of the
// current clip and the given rectangle. Useful for scrollable containers.
// If the intersection is empty or the rectangle has non-positive dimensions,
// returns a painter with an empty clip (no output will be rendered).
func (p *Painter) WithClip(rect Rect) *Painter {
	// Calculate intersection of current clip and new rect
	left := max(p.clip.X, rect.X)
	top := max(p.clip.Y, rect.Y)
	right := min(p.clip.X+p.clip.W, rect.X+rect.W)
	bottom := min(p.clip.Y+p.clip.H, rect.Y+rect.H)

	// If no valid intersection, return painter with zero-size clip
	if left >= right || top >= bottom {
		return &Painter{
			buf:        p.buf,
			clip:       Rect{},
			gp:         p.gp,
			widgetRect: p.widgetRect,
			paneRect:   p.paneRect,
			screenW:    p.screenW,
			screenH:    p.screenH,
			time:       p.time,
		}
	}

	return &Painter{
		buf: p.buf,
		clip: Rect{
			X: left,
			Y: top,
			W: right - left,
			H: bottom - top,
		},
		gp:         p.gp,
		widgetRect: p.widgetRect,
		paneRect:   p.paneRect,
		screenW:    p.screenW,
		screenH:    p.screenH,
		time:       p.time,
	}
}

// SetWidgetRect sets the widget rectangle for dynamic color context.
func (p *Painter) SetWidgetRect(r Rect) { p.widgetRect = r }

// SetPaneRect sets the pane rectangle for dynamic color context.
func (p *Painter) SetPaneRect(r Rect) { p.paneRect = r }

// SetScreenSize sets the screen dimensions for dynamic color context.
func (p *Painter) SetScreenSize(w, h int) { p.screenW = w; p.screenH = h }

// SetTime sets the animation time for dynamic color context.
func (p *Painter) SetTime(t float32) { p.time = t }

// Time returns the current animation time.
func (p *Painter) Time() float32 { return p.time }

// HasAnimations reports whether any drawn dynamic color was animated.
func (p *Painter) HasAnimations() bool { return p.hasAnim }

// MarkAnimated signals that this frame contains animated content and the
// framework should schedule a repaint. Widgets that resolve DynamicColors
// themselves (via SetCell instead of SetDynamicCell) should call this when
// any resolved color was animated.
func (p *Painter) MarkAnimated() { p.hasAnim = true }

// SetDynamicCell writes a cell using a DynamicStyle, resolving colors from context.
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

	if ds.FG.IsAnimated() || ds.BG.IsAnimated() {
		p.hasAnim = true
	}

	// Fast path: both static
	if ds.FG.IsStatic() && ds.BG.IsStatic() {
		style := tcell.StyleDefault.Foreground(ds.FG.Resolve(color.ColorContext{})).
			Background(ds.BG.Resolve(color.ColorContext{}))
		if ds.Attrs != 0 {
			style = style.Attributes(ds.Attrs)
		}
		p.buf[y][x] = Cell{Ch: ch, Style: style}
		return
	}

	// Use widgetRect if set, otherwise fall back to clip rect.
	// This allows widgets to work without explicitly calling SetWidgetRect.
	wr := p.widgetRect
	if wr.W == 0 && wr.H == 0 {
		wr = p.clip
	}
	ctx := color.ColorContext{
		X: x - wr.X, Y: y - wr.Y,
		W: wr.W, H: wr.H,
		PX: x - p.paneRect.X, PY: y - p.paneRect.Y,
		PW: p.paneRect.W, PH: p.paneRect.H,
		SX: x, SY: y,
		SW: max(p.screenW, len(p.buf[0])), SH: max(p.screenH, len(p.buf)),
		T: p.time,
	}

	fg := ds.FG.Resolve(ctx)
	bg := ds.BG.Resolve(ctx)
	style := tcell.StyleDefault.Foreground(fg).Background(bg)
	if ds.Attrs != 0 {
		style = style.Attributes(ds.Attrs)
	}
	p.buf[y][x] = Cell{Ch: ch, Style: style}
}

// SetDynamicCellKeepBG writes a character with dynamic FG but preserves existing BG.
// Used by transparent widgets to overlay text on a parent's background/gradient.
func (p *Painter) SetDynamicCellKeepBG(x, y int, ch rune, ds color.DynamicStyle) {
	if p.buf == nil {
		return
	}
	if x < p.clip.X || y < p.clip.Y || x >= p.clip.X+p.clip.W || y >= p.clip.Y+p.clip.H {
		return
	}
	if y < 0 || y >= len(p.buf) || x < 0 || x >= len(p.buf[y]) {
		return
	}

	if ds.FG.IsAnimated() {
		p.hasAnim = true
	}

	// Resolve FG, keep existing BG
	wr := p.widgetRect
	if wr.W == 0 && wr.H == 0 {
		wr = p.clip
	}
	ctx := color.ColorContext{
		X: x - wr.X, Y: y - wr.Y,
		W: wr.W, H: wr.H,
		PX: x - p.paneRect.X, PY: y - p.paneRect.Y,
		PW: p.paneRect.W, PH: p.paneRect.H,
		SX: x, SY: y,
		SW: max(p.screenW, len(p.buf[0])), SH: max(p.screenH, len(p.buf)),
		T: p.time,
	}

	fg := ds.FG.Resolve(ctx)
	_, existingBG, _ := p.buf[y][x].Style.Decompose()
	style := tcell.StyleDefault.Foreground(fg).Background(existingBG)
	if ds.Attrs != 0 {
		style = style.Attributes(ds.Attrs)
	}
	p.buf[y][x] = Cell{Ch: ch, Style: style}
}

// DrawDynamicTextKeepBG draws text with dynamic FG preserving existing BG.
func (p *Painter) DrawDynamicTextKeepBG(x, y int, s string, ds color.DynamicStyle) {
	xx := x
	for _, r := range s {
		p.SetDynamicCellKeepBG(xx, y, r, ds)
		xx++
	}
}

// FillDynamic fills a rectangle using a DynamicStyle.
func (p *Painter) FillDynamic(rect Rect, ch rune, ds color.DynamicStyle) {
	for yy := rect.Y; yy < rect.Y+rect.H; yy++ {
		for xx := rect.X; xx < rect.X+rect.W; xx++ {
			p.SetDynamicCell(xx, yy, ch, ds)
		}
	}
}

// DrawDynamicText draws a string using a DynamicStyle.
func (p *Painter) DrawDynamicText(x, y int, s string, ds color.DynamicStyle) {
	xx := x
	for _, r := range s {
		p.SetDynamicCell(xx, y, r, ds)
		xx++
	}
}
