package widgets

import (
	"github.com/gdamore/tcell/v2"
	"github.com/framegrace/texelui/color"
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
)

// Alignment specifies how text is aligned within a widget.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// Label displays static text with configurable alignment.
// It's a non-interactive widget by default (not focusable).
type Label struct {
	core.BaseWidget
	Text  string
	Style color.DynamicStyle
	Align Alignment

	// Invalidation callback
	inv func(core.Rect)
}

// NewLabel creates a label with the given text.
// Position defaults to 0,0 and size auto-fits the text.
// Use SetPosition and Resize to adjust after adding to a layout.
func NewLabel(text string) *Label {
	l := &Label{
		Text:  text,
		Align: AlignLeft,
	}

	// Get default style from theme
	tm := theme.Get()
	fg := tm.GetSemanticColor("text.primary")
	bg := tm.GetSemanticColor("bg.surface")
	l.Style = color.DynamicStyle{
		FG: color.Solid(fg),
		BG: color.Solid(bg),
	}

	// Auto-size to fit text
	l.Resize(len(text), 1)

	// Labels are not focusable by default
	l.SetFocusable(false)

	return l
}

// Draw renders the label text with the configured alignment.
func (l *Label) Draw(painter *core.Painter) {
	ds := l.Style
	if l.IsFocused() {
		ds.Attrs |= tcell.AttrBold
	}

	// Fill background
	if !l.Transparent {
		painter.FillDynamic(core.Rect{X: l.Rect.X, Y: l.Rect.Y, W: l.Rect.W, H: l.Rect.H}, ' ', ds)
	}

	if l.Text == "" {
		return
	}

	// Calculate text position based on alignment
	textLen := len(l.Text)
	if textLen > l.Rect.W {
		textLen = l.Rect.W
	}

	var startX int
	switch l.Align {
	case AlignLeft:
		startX = l.Rect.X
	case AlignCenter:
		startX = l.Rect.X + (l.Rect.W-textLen)/2
	case AlignRight:
		startX = l.Rect.X + l.Rect.W - textLen
	}

	// Render text (center vertically if height > 1)
	y := l.Rect.Y + l.Rect.H/2
	painter.DrawDynamicText(startX, y, l.Text, ds)
}

// SetInvalidator allows the UI manager to inject a dirty-region invalidator.
func (l *Label) SetInvalidator(fn func(core.Rect)) { l.inv = fn }

// invalidate marks the widget as needing redraw.
func (l *Label) invalidate() {
	if l.inv != nil {
		l.inv(l.Rect)
	}
}
