package widgets

import (
	"unicode/utf8"

	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

// ToggleButton is a compact clickable indicator that shows on/off state.
// Designed for status bar mode indicators (e.g., "TFM", "WRP", "INS").
// Active state renders with reversed colors (FG/BG swapped).
// Disabled buttons render with faded colors and ignore clicks.
type ToggleButton struct {
	core.BaseWidget
	Label    string
	Active   bool
	Disabled bool
	OnToggle func(active bool)
	Style    tcell.Style

	// Invalidation callback
	inv func(core.Rect)
}

// NewToggleButton creates a toggle button with the given label.
// Width equals len(label), height is 1. Not focusable.
func NewToggleButton(label string) *ToggleButton {
	tb := &ToggleButton{
		Label: label,
	}

	// Get default style from theme
	tm := theme.Get()
	fg := tm.GetSemanticColor("text.primary")
	bg := tm.GetSemanticColor("bg.surface")

	// Fall back to White/Black if theme returns default colors
	if fg == tcell.ColorDefault {
		fg = tcell.ColorWhite
	}
	if bg == tcell.ColorDefault {
		bg = tcell.ColorBlack
	}

	tb.Style = tcell.StyleDefault.Foreground(fg).Background(bg)

	tb.SetPosition(0, 0)
	tb.Resize(utf8.RuneCountInString(label), 1)
	tb.SetFocusable(false)

	return tb
}

// Draw renders the toggle button label with normal, reversed, or faded style.
// Disabled+Active shows a faded reversed style (visible but non-interactive).
func (tb *ToggleButton) Draw(painter *core.Painter) {
	style := tb.EffectiveStyle(tb.Style)
	fg, bg, attr := style.Decompose()
	if tb.Active {
		fg, bg = bg, fg
	}
	if tb.Disabled {
		fg = fadeColor(fg, bg, 0.35)
		attr = 0
	}
	style = tcell.StyleDefault.Foreground(fg).Background(bg).Attributes(attr)
	painter.Fill(core.Rect{X: tb.Rect.X, Y: tb.Rect.Y, W: tb.Rect.W, H: 1}, ' ', style)
	painter.DrawText(tb.Rect.X, tb.Rect.Y, tb.Label, style)
}

// fadeColor blends fg toward bg by ratio (0 = fg, 1 = bg).
func fadeColor(fg, bg tcell.Color, ratio float64) tcell.Color {
	fr, ffg, fb := fg.RGB()
	br, bbg, bb := bg.RGB()
	mix := func(a, b int32, r float64) int32 {
		return a + int32(float64(b-a)*r)
	}
	return tcell.NewRGBColor(mix(fr, br, ratio), mix(ffg, bbg, ratio), mix(fb, bb, ratio))
}

// HandleMouse processes mouse input. Left click toggles the active state.
// Disabled buttons ignore all mouse input.
func (tb *ToggleButton) HandleMouse(ev *tcell.EventMouse) bool {
	if tb.Disabled {
		return false
	}
	x, y := ev.Position()
	if !tb.HitTest(x, y) {
		return false
	}

	switch ev.Buttons() {
	case tcell.Button1:
		tb.toggle()
		return true
	}

	return false
}

// toggle switches the active state and fires the OnToggle callback.
func (tb *ToggleButton) toggle() {
	tb.Active = !tb.Active
	tb.invalidate()
	if tb.OnToggle != nil {
		tb.OnToggle(tb.Active)
	}
}

// SetInvalidator allows the UI manager to inject a dirty-region invalidator.
func (tb *ToggleButton) SetInvalidator(fn func(core.Rect)) { tb.inv = fn }

// invalidate marks the widget as needing redraw.
func (tb *ToggleButton) invalidate() {
	if tb.inv != nil {
		tb.inv(tb.Rect)
	}
}
