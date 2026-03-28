package widgets

import (
	"github.com/gdamore/tcell/v2"
	"github.com/framegrace/texelui/color"
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
)

// Input is a single-line text entry widget with horizontal scrolling.
type Input struct {
	core.BaseWidget
	Text       string
	CaretPos   int // Caret position in runes (0 to len([]rune(Text)))
	OffX       int // Horizontal scroll offset
	Style      color.DynamicStyle
	CaretStyle color.DynamicStyle

	// Optional placeholder text shown when empty
	Placeholder string

	// Optional validation/change callback
	OnChange func(text string)
	// Optional submit callback (Enter key)
	OnSubmit func(text string)
	// Optional blur callback
	OnBlur func(text string)

	// Mouse state
	mouseDown bool

	// Insert vs replace mode: false=insert (default), true=replace (overwrite)
	replaceMode bool

	// Invalidation callback
	inv func(core.Rect)
}

// NewInput creates a single-line input field.
// Position defaults to 0,0 and width to 20.
// Use SetPosition and Resize to adjust after adding to a layout.
func NewInput() *Input {
	tm := theme.Get()
	bg := tm.GetSemanticColor("bg.surface")
	fg := tm.GetSemanticColor("text.primary")
	caret := tm.GetSemanticColor("caret")

	i := &Input{
		Text:     "",
		CaretPos: 0,
		Style: color.DynamicStyle{
			FG: color.Solid(fg),
			BG: color.Solid(bg),
		},
		CaretStyle: color.DynamicStyle{
			FG: color.Solid(caret),
		},
	}

	// Configure focused style
	i.SetFocusedStyle(tcell.StyleDefault.Foreground(fg).Background(bg), true)

	i.Resize(20, 1) // Default width, always single-line
	i.SetFocusable(true)

	return i
}

// SetInvalidator allows the UI manager to inject a dirty-region invalidator.
func (i *Input) SetInvalidator(fn func(core.Rect)) { i.inv = fn }

// GetKeyHints implements core.KeyHintsProvider.
func (i *Input) GetKeyHints() []core.KeyHint {
	return []core.KeyHint{
		{Key: "←→", Label: "Move"},
		{Key: "↑↓", Label: "Navigate"},
	}
}

// Blur removes focus and triggers the OnBlur callback if set.
func (i *Input) Blur() {
	wasFocused := i.IsFocused()
	i.BaseWidget.Blur()
	if wasFocused && i.OnBlur != nil {
		i.OnBlur(i.Text)
	}
}

// Draw renders the input field with text and caret.
func (i *Input) Draw(painter *core.Painter) {
	ds := i.Style
	focused := i.IsFocused()

	// When focused, add underline to show the input field extent
	if focused {
		ds.Attrs |= tcell.AttrUnderline
	}

	// Fill background with underline when focused
	if !i.Transparent {
		painter.FillDynamic(core.Rect{X: i.Rect.X, Y: i.Rect.Y, W: i.Rect.W, H: 1}, ' ', ds)
	}

	// Determine what to display
	displayText := i.Text
	if displayText == "" && i.Placeholder != "" && !focused {
		// Show placeholder in dimmed color when not focused and empty
		ctx := color.ColorContext{}
		bg := ds.BG.Resolve(ctx)
		placeholderStyle := color.DynamicStyle{
			FG: color.Solid(tcell.ColorGray),
			BG: color.Solid(bg),
		}
		if i.Transparent {
			painter.DrawDynamicTextKeepBG(i.Rect.X, i.Rect.Y, i.Placeholder, placeholderStyle)
		} else {
			painter.DrawDynamicText(i.Rect.X, i.Rect.Y, i.Placeholder, placeholderStyle)
		}
		return
	}

	// Ensure caret is visible (adjust scroll offset)
	i.ensureCaretVisible()

	// Convert to runes for proper unicode handling
	runes := []rune(displayText)

	// Render visible portion of text
	x := i.Rect.X
	drawText := painter.DrawDynamicText
	if i.Transparent {
		drawText = painter.DrawDynamicTextKeepBG
	}
	for idx := i.OffX; idx < len(runes) && x < i.Rect.X+i.Rect.W; idx++ {
		drawText(x, i.Rect.Y, string(runes[idx]), ds)
		x++
	}

	// Draw caret if focused
	if focused {
		caretX := i.Rect.X + i.CaretPos - i.OffX
		if caretX >= i.Rect.X && caretX < i.Rect.X+i.Rect.W {
			// Determine what character is under the caret
			ch := ' '
			if i.CaretPos >= 0 && i.CaretPos < len(runes) {
				ch = runes[i.CaretPos]
			}

			// Determine caret style based on mode — resolve colors for swap
			ctx := color.ColorContext{}
			fg := ds.FG.Resolve(ctx)
			bg := ds.BG.Resolve(ctx)
			var caretDS color.DynamicStyle
			if i.replaceMode {
				// Underline caret in replace mode
				caretDS = color.DynamicStyle{
					FG:    color.Solid(fg),
					BG:    color.Solid(bg),
					Attrs: tcell.AttrUnderline,
				}
			} else {
				// Reverse video caret in insert mode
				caretDS = color.DynamicStyle{
					FG: color.Solid(bg),
					BG: color.Solid(fg),
				}
			}

			// Draw the character with caret styling
			painter.SetDynamicCell(caretX, i.Rect.Y, ch, caretDS)
		}
	}
}

// ensureCaretVisible adjusts scroll offset to keep caret in view.
func (i *Input) ensureCaretVisible() {
	if i.CaretPos < i.OffX {
		i.OffX = i.CaretPos
	}
	if i.CaretPos >= i.OffX+i.Rect.W {
		i.OffX = i.CaretPos - i.Rect.W + 1
	}
	if i.OffX < 0 {
		i.OffX = 0
	}
}

// HandleKey processes keyboard input for text editing.
func (i *Input) HandleKey(ev *tcell.EventKey) bool {
	runes := []rune(i.Text)
	textLen := len(runes)

	switch ev.Key() {
	case tcell.KeyLeft:
		if i.CaretPos > 0 {
			i.CaretPos--
		}
		i.invalidate()
		return true

	case tcell.KeyRight:
		if i.CaretPos < textLen {
			i.CaretPos++
		}
		i.invalidate()
		return true

	case tcell.KeyHome:
		i.CaretPos = 0
		i.invalidate()
		return true

	case tcell.KeyEnd:
		i.CaretPos = textLen
		i.invalidate()
		return true

	case tcell.KeyEnter:
		// Submit the input - triggers OnSubmit callback and signals handled
		// so UIManager can advance focus if AdvanceFocusOnEnter is enabled
		if i.OnSubmit != nil {
			i.OnSubmit(i.Text)
		}
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if i.CaretPos > 0 {
			runes = append(runes[:i.CaretPos-1], runes[i.CaretPos:]...)
			i.CaretPos--
			i.Text = string(runes)
			i.onChange()
			i.invalidate()
		}
		return true

	case tcell.KeyDelete:
		if i.CaretPos < textLen {
			runes = append(runes[:i.CaretPos], runes[i.CaretPos+1:]...)
			i.Text = string(runes)
			i.onChange()
			i.invalidate()
		}
		return true

	case tcell.KeyInsert:
		// Toggle insert/replace mode
		i.replaceMode = !i.replaceMode
		i.invalidate()
		return true

	case tcell.KeyRune:
		// Insert or replace character at caret position
		r := ev.Rune()
		if i.replaceMode && i.CaretPos < textLen {
			// Overwrite current character
			runes[i.CaretPos] = r
			i.CaretPos++
			i.Text = string(runes)
		} else {
			// Insert mode (default)
			runes = append(runes[:i.CaretPos], append([]rune{r}, runes[i.CaretPos:]...)...)
			i.CaretPos++
			i.Text = string(runes)
		}
		i.onChange()
		i.invalidate()
		return true
	}

	return false
}

// HandleMouse processes mouse input for caret positioning.
func (i *Input) HandleMouse(ev *tcell.EventMouse) bool {
	x, y := ev.Position()
	if !i.HitTest(x, y) {
		i.mouseDown = false
		return false
	}

	switch ev.Buttons() {
	case tcell.Button1: // Left mouse button
		i.mouseDown = true
		// Position caret at click location
		clickX := x - i.Rect.X + i.OffX
		runes := []rune(i.Text)
		if clickX < 0 {
			i.CaretPos = 0
		} else if clickX >= len(runes) {
			i.CaretPos = len(runes)
		} else {
			i.CaretPos = clickX
		}
		i.invalidate()
		return true

	case tcell.ButtonNone: // Mouse release
		i.mouseDown = false
	}

	return false
}

// onChange triggers the OnChange callback if set.
func (i *Input) onChange() {
	if i.OnChange != nil {
		i.OnChange(i.Text)
	}
}

// invalidate marks the widget as needing redraw.
func (i *Input) invalidate() {
	if i.inv != nil {
		i.inv(i.Rect)
	}
}
