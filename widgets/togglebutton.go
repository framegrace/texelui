package widgets

import (
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

// ToggleButton is a compact clickable indicator that shows on/off state.
// Designed for status bar mode indicators (e.g., "TFM", "WRP", "INS").
// Active state renders with reversed colors (FG/BG swapped).
type ToggleButton struct {
	core.BaseWidget
	Label    string
	Active   bool
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
	tb.Resize(len(label), 1)
	tb.SetFocusable(false)

	return tb
}

// Draw renders the toggle button label with normal or reversed style.
func (tb *ToggleButton) Draw(painter *core.Painter) {
	style := tb.EffectiveStyle(tb.Style)
	if tb.Active {
		fg, bg, attr := style.Decompose()
		style = tcell.StyleDefault.Foreground(bg).Background(fg).Attributes(attr)
	}
	painter.Fill(core.Rect{X: tb.Rect.X, Y: tb.Rect.Y, W: tb.Rect.W, H: 1}, ' ', style)
	painter.DrawText(tb.Rect.X, tb.Rect.Y, tb.Label, style)
}

// HandleMouse processes mouse input. Left click toggles the active state.
func (tb *ToggleButton) HandleMouse(ev *tcell.EventMouse) bool {
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
