package widgets

import (
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

// Link is a clickable text widget styled with underline.
// Used for web links in the browser. Text is always underlined;
// focus adds reverse/highlight but keeps the underline.
type Link struct {
	core.BaseWidget
	Text    string
	Style   tcell.Style
	OnClick func()

	// Invalidation callback
	inv func(core.Rect)
}

// NewLink creates a link with the given text.
// Position defaults to 0,0 and size auto-fits the text length.
// Use SetPosition and Resize to adjust after adding to a layout.
func NewLink(text string) *Link {
	l := &Link{
		Text: text,
	}

	// Get default style from theme — use accent color with underline
	tm := theme.Get()
	fg := tm.GetSemanticColor("accent.primary")
	bg := tm.GetSemanticColor("bg.surface")
	l.Style = tcell.StyleDefault.Foreground(fg).Background(bg).Underline(true)

	// Configure focused style — reverse colors but keep underline
	focusFg := tm.GetSemanticColor("text.inverse")
	focusBg := tm.GetSemanticColor("border.focus")
	l.SetFocusedStyle(tcell.StyleDefault.Foreground(focusFg).Background(focusBg).Underline(true), true)

	// Auto-size to fit text
	l.Resize(len(text), 1)

	// Links are focusable by default
	l.SetFocusable(true)

	return l
}

// Draw renders the link text with underline style.
func (l *Link) Draw(painter *core.Painter) {
	style := l.EffectiveStyle(l.Style)

	// Ensure underline is always applied, even after EffectiveStyle merges attributes
	style = style.Underline(true)

	// Fill background
	painter.Fill(core.Rect{X: l.Rect.X, Y: l.Rect.Y, W: l.Rect.W, H: l.Rect.H}, ' ', style)

	if l.Text == "" {
		return
	}

	// Truncate if too long
	displayText := l.Text
	if len(displayText) > l.Rect.W {
		displayText = displayText[:l.Rect.W]
	}

	// Draw text at position
	painter.DrawText(l.Rect.X, l.Rect.Y+l.Rect.H/2, displayText, style)
}

// HandleKey processes keyboard input. Enter triggers the OnClick callback.
func (l *Link) HandleKey(ev *tcell.EventKey) bool {
	if ev.Key() == tcell.KeyEnter {
		l.activate()
		return true
	}
	return false
}

// HandleMouse processes mouse input. Left click triggers the OnClick callback.
func (l *Link) HandleMouse(ev *tcell.EventMouse) bool {
	if ev.Buttons()&tcell.Button1 != 0 {
		l.activate()
		return true
	}
	return false
}

// activate triggers the OnClick callback if set.
func (l *Link) activate() {
	if l.OnClick != nil {
		l.OnClick()
	}
}

// SetInvalidator allows the UI manager to inject a dirty-region invalidator.
func (l *Link) SetInvalidator(fn func(core.Rect)) { l.inv = fn }

// GetKeyHints implements core.KeyHintsProvider.
func (l *Link) GetKeyHints() []core.KeyHint {
	return []core.KeyHint{
		{Key: "Enter", Label: "Follow link"},
	}
}

// invalidate marks the widget as needing redraw.
func (l *Link) invalidate() {
	if l.inv != nil {
		l.inv(l.Rect)
	}
}
