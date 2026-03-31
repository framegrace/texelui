// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: texelui/primitives/tabbar.go
// Summary: Horizontal tab bar widget with keyboard and mouse navigation.

package primitives

import (
	"github.com/framegrace/texelui/color"
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

// tabEditor is a minimal inline text editor used by TabBar for tab renaming.
// It avoids a circular import between primitives and widgets.
type tabEditor struct {
	text     []rune
	caretPos int // caret position in runes (0..len(text))
	offX     int // horizontal scroll offset

	// OnSubmit is called when the user presses Enter.
	OnSubmit func(text string)
}

// newTabEditor creates a new tabEditor pre-filled with initial text.
func newTabEditor(initial string) *tabEditor {
	runes := []rune(initial)
	return &tabEditor{
		text:     runes,
		caretPos: len(runes),
	}
}

// Text returns the current editor content as a string.
func (e *tabEditor) Text() string { return string(e.text) }

// HandleKey processes keyboard input. Returns true if handled.
// Enter triggers OnSubmit; Escape is NOT handled here (caller handles it).
func (e *tabEditor) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyLeft:
		if e.caretPos > 0 {
			e.caretPos--
		}
		return true

	case tcell.KeyRight:
		if e.caretPos < len(e.text) {
			e.caretPos++
		}
		return true

	case tcell.KeyHome:
		e.caretPos = 0
		return true

	case tcell.KeyEnd:
		e.caretPos = len(e.text)
		return true

	case tcell.KeyEnter:
		if e.OnSubmit != nil {
			e.OnSubmit(e.Text())
		}
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if e.caretPos > 0 {
			e.text = append(e.text[:e.caretPos-1], e.text[e.caretPos:]...)
			e.caretPos--
		}
		return true

	case tcell.KeyDelete:
		if e.caretPos < len(e.text) {
			e.text = append(e.text[:e.caretPos], e.text[e.caretPos+1:]...)
		}
		return true

	case tcell.KeyRune:
		r := ev.Rune()
		e.text = append(e.text[:e.caretPos], append([]rune{r}, e.text[e.caretPos:]...)...)
		e.caretPos++
		return true
	}

	return false
}

// Draw renders the editor into the painter at the given position/width.
// Uses the provided style for text; reverses colors for the caret.
func (e *tabEditor) Draw(painter *core.Painter, x, y, w int, style tcell.Style) {
	if w <= 0 {
		return
	}

	// Adjust scroll offset to keep caret visible
	if e.caretPos < e.offX {
		e.offX = e.caretPos
	}
	if e.caretPos >= e.offX+w {
		e.offX = e.caretPos - w + 1
	}
	if e.offX < 0 {
		e.offX = 0
	}

	fg, bg, attrs := style.Decompose()

	// Draw text
	for col := 0; col < w; col++ {
		idx := e.offX + col
		ch := ' '
		if idx < len(e.text) {
			ch = e.text[idx]
		}
		cellStyle := tcell.StyleDefault.Foreground(fg).Background(bg).Attributes(attrs)
		// Caret: reverse video
		if idx == e.caretPos {
			cellStyle = tcell.StyleDefault.Foreground(bg).Background(fg)
		}
		painter.SetCell(x+col, y, ch, cellStyle)
	}
}

// Powerline separator characters (Nerd Font Private Use Area).
const (
	plLeftTriangle  = '\ue0ba'
	plRightTriangle = '\ue0b8'
	plLeftThinLine  = '\ue0bb'
	plRightThinLine = '\ue0b9'
	blendChar       = '\u2594' // Upper one-eighth block (thin accent line)
)

// TabBarStyle controls the colors used by the tab bar.
// Zero-value fields are resolved from the theme at draw time.
type TabBarStyle struct {
	ActiveBG   color.DynamicColor
	ActiveFG   color.DynamicColor
	InactiveBG color.DynamicColor
	InactiveFG color.DynamicColor
	BarBG      color.DynamicColor
	ContentBG  color.DynamicColor
	NoBlendRow bool // No blend row at all (height=1)
}

// TabItem represents a single tab in a TabBar.
type TabItem struct {
	Label string
	ID    string             // Optional identifier for the tab
	Color color.DynamicColor // Optional per-tab accent color (zero = use style defaults)
}

// TabBar is a horizontal tab navigation widget.
// It displays tabs and handles keyboard (Left/Right, number keys) and mouse navigation.
type TabBar struct {
	core.BaseWidget
	Tabs      []TabItem
	ActiveIdx int
	OnChange  func(int) // Called when active tab changes

	// Edit mode callbacks
	OnRename     func(index int, newName string) // Called when edit confirmed via Enter
	OnEditCancel func(index int)                 // Called when edit cancelled via Escape

	// Styling
	Style           TabBarStyle
	ShowFocusMarker bool // Show '►' marker when focused (default true)

	// Focus navigation callback — called when Up/Down should cycle focus
	// out of the tab bar. Set by TabLayout to wire into CycleFocus.
	OnFocusExit func(forward bool)

	// Mouse hover state
	hoverIdx int // Index of tab under mouse cursor (-1 if none)

	// Edit mode state
	editIdx      int        // Index being edited; -1 when not editing
	editInput    *tabEditor // Inline text editor for renaming
	editOriginal string     // Original label before edit started

	inv func(core.Rect)
}

// TabBarHeight returns the height needed by the tab bar.
// Returns 2 when the blend row is enabled (default), 1 otherwise.
func (tb *TabBar) TabBarHeight() int {
	if tb.Style.NoBlendRow {
		return 1
	}
	return 2
}

// NewTabBar creates a new tab bar at the specified position.
// Width determines the available space for tabs. Height is 2 by default
// (1 tab row + 1 blend row), or 1 if Style.NoBlendRow is set.
func NewTabBar(x, y, w int, tabs []TabItem) *TabBar {
	tb := &TabBar{
		Tabs:            tabs,
		ActiveIdx:       0,
		ShowFocusMarker: true,
		hoverIdx:        -1,
		editIdx:         -1,
	}

	tb.SetPosition(x, y)
	tb.Resize(w, tb.TabBarHeight())
	tb.SetFocusable(true)

	// Configure focus style from theme
	tm := theme.Get()
	fg := tm.GetSemanticColor("text.primary")
	bg := tm.GetSemanticColor("bg.surface")
	tb.SetFocusedStyle(tcell.StyleDefault.Foreground(fg).Background(bg), true)

	return tb
}

// SetInvalidator allows the UI manager to inject a dirty-region invalidator.
func (tb *TabBar) SetInvalidator(fn func(core.Rect)) {
	tb.inv = fn
}

// SetActive changes the active tab by index.
func (tb *TabBar) SetActive(idx int) {
	if idx < 0 || idx >= len(tb.Tabs) {
		return
	}
	if idx == tb.ActiveIdx {
		return
	}
	tb.ActiveIdx = idx
	tb.invalidate()
	if tb.OnChange != nil {
		tb.OnChange(idx)
	}
}

// ActiveTab returns the currently active tab item.
func (tb *TabBar) ActiveTab() TabItem {
	if tb.ActiveIdx >= 0 && tb.ActiveIdx < len(tb.Tabs) {
		return tb.Tabs[tb.ActiveIdx]
	}
	return TabItem{}
}

// Draw renders the tab bar with powerline-style separators.
func (tb *TabBar) Draw(painter *core.Painter) {
	if len(tb.Tabs) == 0 {
		return
	}

	// Resolve style DynamicColors, falling back to theme defaults for unset fields.
	tm := theme.Get()
	dynResolve := func(dc color.DynamicColor, fallback string) color.DynamicColor {
		if dc.IsZero() {
			return color.Solid(tm.GetSemanticColor(fallback))
		}
		return dc
	}
	activeBG := dynResolve(tb.Style.ActiveBG, "accent")
	activeFG := dynResolve(tb.Style.ActiveFG, "text.inverse")
	inactiveBG := dynResolve(tb.Style.InactiveBG, "bg.surface")
	barBG := dynResolve(tb.Style.BarBG, "bg.mantle")
	contentBG := dynResolve(tb.Style.ContentBG, "bg.surface")

	// If any style color is dynamic (non-static), mark the painter as animated
	// so the framework auto-refreshes.
	if !activeBG.IsStatic() || !activeFG.IsStatic() ||
		!inactiveBG.IsStatic() || !barBG.IsStatic() {
		painter.MarkAnimated()
	}

	focused := tb.IsFocused()
	x := tb.Rect.X
	y := tb.Rect.Y
	maxX := tb.Rect.X + tb.Rect.W

	// Build DynamicStyles for each visual element.
	activeDS := color.DynamicStyle{FG: activeFG, BG: activeBG}
	if focused {
		activeDS.Attrs = tcell.AttrBold
	}
	barDS := color.DynamicStyle{FG: barBG, BG: barBG}

	// Inactive tabs: normal text when bar unfocused, accent FG when bar focused.
	inactFG := color.Solid(tm.GetSemanticColor("text.primary"))
	if focused {
		inactFG = activeBG
	}

	// tabDynBG returns the DynamicColor for tab i's background.
	tabDynBG := func(i int) color.DynamicColor {
		if i == tb.ActiveIdx {
			return activeBG
		}
		if i >= 0 && i < len(tb.Tabs) && !tb.Tabs[i].Color.IsZero() {
			return tb.Tabs[i].Color
		}
		return inactiveBG
	}

	// Row 0: powerline tab row
	// Leading left triangle: FG = first tab's BG, BG = barBG
	if x < maxX {
		painter.SetDynamicCell(x, y, plLeftTriangle, color.DynamicStyle{FG: tabDynBG(0), BG: barBG})
		x++
	}

	for i, tab := range tb.Tabs {
		tabLabel := " " + tab.Label + " "
		isActive := i == tb.ActiveIdx
		isHover := i == tb.hoverIdx && !isActive

		var ds color.DynamicStyle
		if isActive {
			ds = activeDS
		} else {
			ds = color.DynamicStyle{FG: inactFG, BG: tabDynBG(i)}
			if isHover {
				ds.Attrs = tcell.AttrReverse
			}
		}

		// Draw tab label characters (or inline editor when editing this tab)
		if tb.editIdx == i && tb.editInput != nil {
			if x < maxX {
				painter.SetDynamicCell(x, y, ' ', ds)
				x++
			}
			editLen := len([]rune(tb.editInput.Text())) + 1
			labelLen := len([]rune(tab.Label))
			labelWidth := editLen
			if labelLen > labelWidth {
				labelWidth = labelLen
			}
			if labelWidth < 1 {
				labelWidth = 1
			}
			inputW := labelWidth
			if x+inputW > maxX {
				inputW = maxX - x
			}
			if inputW > 0 {
				// Editor still uses static style (it draws its own cells)
				ctx := color.ColorContext{T: painter.Time()}
				editStyle := tcell.StyleDefault.Foreground(ds.FG.Resolve(ctx)).Background(ds.BG.Resolve(ctx))
				if ds.Attrs != 0 {
					editStyle = editStyle.Attributes(ds.Attrs)
				}
				tb.editInput.Draw(painter, x, y, inputW, editStyle)
				x += inputW
			}
			if x < maxX {
				painter.SetDynamicCell(x, y, ' ', ds)
				x++
			}
		} else {
			for _, ch := range tabLabel {
				if x >= maxX {
					break
				}
				painter.SetDynamicCell(x, y, ch, ds)
				x++
			}
		}

		// Powerline separator between tabs
		if i < len(tb.Tabs)-1 && x < maxX {
			curBG := tabDynBG(i)
			nextBG := tabDynBG(i + 1)
			if i < tb.ActiveIdx {
				painter.SetDynamicCell(x, y, plLeftTriangle, color.DynamicStyle{FG: nextBG, BG: curBG})
			} else {
				painter.SetDynamicCell(x, y, plRightTriangle, color.DynamicStyle{FG: curBG, BG: nextBG})
			}
			x++
		}
	}

	// Trailing right triangle
	if x < maxX {
		painter.SetDynamicCell(x, y, plRightTriangle, color.DynamicStyle{FG: tabDynBG(len(tb.Tabs) - 1), BG: barBG})
		x++
	}

	tabsEndX := x

	// Fill rest of row 0 with bar background
	for x < maxX {
		painter.SetDynamicCell(x, y, ' ', barDS)
		x++
	}

	// Row 1: blend line — accent under tabs, then gradient fade to content BG.
	if !tb.Style.NoBlendRow && tb.Rect.H >= 2 {
		blendY := y + 1
		// Under tabs: use activeBG directly so Pulse descriptors propagate.
		accentDS := color.DynamicStyle{FG: activeBG, BG: contentBG}
		for bx := tb.Rect.X; bx < tabsEndX && bx < maxX; bx++ {
			painter.SetDynamicCell(bx, blendY, blendChar, accentDS)
		}
		// After tabs: gradient fade from accent to content BG (spatial, static).
		fadeW := maxX - tabsEndX
		if fadeW > 0 {
			ctx := color.ColorContext{T: painter.Time()}
			resolvedAccent := activeBG.Resolve(ctx)
			resolvedContent := contentBG.Resolve(ctx)
			fadeFG := color.Linear(0,
				color.Stop(0, resolvedAccent),
				color.Stop(1, resolvedContent),
			).WithLocal().Build()
			fadeDS := color.DynamicStyle{FG: fadeFG, BG: color.Solid(resolvedContent)}
			for bx := tabsEndX; bx < maxX; bx++ {
				painter.SetDynamicCell(bx, blendY, blendChar, fadeDS)
			}
		}
	}
}

// HandleKey processes keyboard input for tab navigation.
// Returns true if the key was handled.
func (tb *TabBar) HandleKey(ev *tcell.EventKey) bool {
	if len(tb.Tabs) == 0 {
		return false
	}

	// When in edit mode, route keys to the inline input widget.
	// Escape cancels; Tab confirms; Enter is handled by input's OnSubmit.
	if tb.IsEditing() {
		switch ev.Key() {
		case tcell.KeyEscape:
			tb.CancelEdit()
			return true
		case tcell.KeyTab:
			// Tab confirms the edit
			tb.confirmEdit(tb.editInput.Text())
			return true
		default:
			handled := tb.editInput.HandleKey(ev)
			if handled {
				tb.invalidate()
			}
			return handled
		}
	}

	switch ev.Key() {
	case tcell.KeyLeft:
		if tb.ActiveIdx > 0 {
			tb.SetActive(tb.ActiveIdx - 1)
			return true
		}
		return false

	case tcell.KeyRight:
		if tb.ActiveIdx < len(tb.Tabs)-1 {
			tb.SetActive(tb.ActiveIdx + 1)
			return true
		}
		return false

	case tcell.KeyHome:
		if tb.ActiveIdx != 0 {
			tb.SetActive(0)
			return true
		}
		return false

	case tcell.KeyEnd:
		lastIdx := len(tb.Tabs) - 1
		if tb.ActiveIdx != lastIdx {
			tb.SetActive(lastIdx)
			return true
		}
		return false

	case tcell.KeyRune:
		// Number keys 1-9 for direct tab selection
		r := ev.Rune()
		if r >= '1' && r <= '9' {
			idx := int(r - '1')
			if idx < len(tb.Tabs) && idx != tb.ActiveIdx {
				tb.SetActive(idx)
				return true
			}
		}
		return false
	}

	return false
}

// HandleMouse processes mouse input for tab selection and hover.
func (tb *TabBar) HandleMouse(ev *tcell.EventMouse) bool {
	if len(tb.Tabs) == 0 {
		return false
	}

	x, y := ev.Position()

	// Check if mouse left the tab bar area
	if !tb.HitTest(x, y) {
		if tb.hoverIdx != -1 {
			tb.hoverIdx = -1
			tb.invalidate()
		}
		return false
	}

	// Calculate which tab the mouse is over
	tabIdx := tb.TabAtX(x)

	// Update hover state (only for non-active tabs)
	if tabIdx != tb.hoverIdx {
		tb.hoverIdx = tabIdx
		tb.invalidate()
	}

	// Handle click for tab selection and edit mode
	if ev.Buttons() == tcell.Button1 {
		if tb.IsEditing() && tabIdx != tb.editIdx {
			// Click outside editing tab confirms the edit
			tb.confirmEdit(tb.editInput.Text())
		}
		if tabIdx >= 0 {
			if tabIdx == tb.ActiveIdx && !tb.IsEditing() {
				// Click on already-active tab enters edit mode
				tb.EditTab(tabIdx)
			} else if tabIdx != tb.ActiveIdx {
				tb.SetActive(tabIdx)
			}
		}
		return true
	}

	return true
}

// TabAtX returns the tab index at the given absolute x position, or -1 if none.
// Layout: [leftTri][" Label "][sep][" Label "][sep]...[rightTri][barBG...]
func (tb *TabBar) TabAtX(x int) int {
	col := tb.Rect.X

	// Skip leading left triangle
	if x == col {
		return -1
	}
	col++

	for i, tab := range tb.Tabs {
		tabWidth := len(" " + tab.Label + " ")

		if x >= col && x < col+tabWidth {
			return i
		}
		col += tabWidth

		// Separator after each tab (except the last, which has trailing triangle)
		if i < len(tb.Tabs)-1 {
			if x == col {
				return -1 // separator
			}
			col++
		}
	}

	// Trailing right triangle or bar fill
	return -1
}

// ClearHover resets the hover state (e.g., when mouse leaves or focus changes).
func (tb *TabBar) ClearHover() {
	if tb.hoverIdx != -1 {
		tb.hoverIdx = -1
		tb.invalidate()
	}
}

// invalidate marks the widget as needing redraw.
func (tb *TabBar) invalidate() {
	if tb.inv != nil {
		tb.inv(tb.Rect)
	}
}

// IsEditing returns whether the tab bar is currently in inline edit mode.
func (tb *TabBar) IsEditing() bool {
	return tb.editIdx >= 0
}

// EditTab enters inline edit mode for the tab at the given index.
// Pre-fills the input with the current label. No-op if index is out of range.
func (tb *TabBar) EditTab(index int) {
	if index < 0 || index >= len(tb.Tabs) {
		return
	}
	tb.editIdx = index
	tb.editOriginal = tb.Tabs[index].Label

	ed := newTabEditor(tb.editOriginal)
	ed.OnSubmit = func(text string) {
		tb.confirmEdit(text)
	}
	tb.editInput = ed
	tb.invalidate()
}

// CancelEdit cancels any active edit, reverting the label to its original value.
func (tb *TabBar) CancelEdit() {
	if tb.editIdx < 0 {
		return
	}
	idx := tb.editIdx
	tb.Tabs[idx].Label = tb.editOriginal
	tb.editIdx = -1
	tb.editInput = nil
	tb.editOriginal = ""
	if tb.OnEditCancel != nil {
		tb.OnEditCancel(idx)
	}
	tb.invalidate()
}

// confirmEdit commits the edited text and exits edit mode.
func (tb *TabBar) confirmEdit(text string) {
	if tb.editIdx < 0 {
		return
	}
	idx := tb.editIdx
	tb.Tabs[idx].Label = text
	tb.editIdx = -1
	tb.editInput = nil
	tb.editOriginal = ""
	if tb.OnRename != nil {
		tb.OnRename(idx, text)
	}
	tb.invalidate()
}

// GetKeyHints implements KeyHintsProvider from core package.
func (tb *TabBar) GetKeyHints() []core.KeyHint {
	return []core.KeyHint{
		{Key: "←→", Label: "Switch"},
		{Key: "1-9", Label: "Jump"},
		{Key: "↓", Label: "Content"},
	}
}
