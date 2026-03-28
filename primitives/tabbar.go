// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: texelui/primitives/tabbar.go
// Summary: Horizontal tab bar widget with keyboard and mouse navigation.

package primitives

import (
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

// Powerline separator characters (Nerd Font Private Use Area).
const (
	plLeftTriangle  = '\ue0ba'
	plRightTriangle = '\ue0b8'
	plLeftThinLine  = '\ue0bb'
	plRightThinLine = '\ue0b9'
	blendChar       = '\u2580' // Upper half block
)

// TabBarStyle controls the colors used by the tab bar.
// Zero-value fields are resolved from the theme at draw time.
type TabBarStyle struct {
	ActiveBG   tcell.Color
	ActiveFG   tcell.Color
	InactiveBG tcell.Color
	InactiveFG tcell.Color
	BarBG      tcell.Color
	ContentBG  tcell.Color
	NoBlendRow bool
}

// TabItem represents a single tab in a TabBar.
type TabItem struct {
	Label string
	ID    string // Optional identifier for the tab
}

// TabBar is a horizontal tab navigation widget.
// It displays tabs and handles keyboard (Left/Right, number keys) and mouse navigation.
type TabBar struct {
	core.BaseWidget
	Tabs      []TabItem
	ActiveIdx int
	OnChange  func(int) // Called when active tab changes

	// Styling
	Style           TabBarStyle
	ShowFocusMarker bool // Show '►' marker when focused (default true)

	// Mouse hover state
	hoverIdx int // Index of tab under mouse cursor (-1 if none)

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

// resolveColors returns TabBarStyle with zero-value colors resolved from theme.
func (tb *TabBar) resolveColors() TabBarStyle {
	s := tb.Style
	tm := theme.Get()
	if s.ActiveBG == 0 {
		s.ActiveBG = tm.GetSemanticColor("accent")
	}
	if s.ActiveFG == 0 {
		s.ActiveFG = tm.GetSemanticColor("text.inverse")
	}
	if s.InactiveBG == 0 {
		s.InactiveBG = tm.GetSemanticColor("bg.surface")
	}
	if s.InactiveFG == 0 {
		s.InactiveFG = tm.GetSemanticColor("text.muted")
	}
	if s.BarBG == 0 {
		s.BarBG = tm.GetSemanticColor("bg.mantle")
	}
	if s.ContentBG == 0 {
		s.ContentBG = tm.GetSemanticColor("bg.surface")
	}
	return s
}

// Draw renders the tab bar with powerline-style separators.
func (tb *TabBar) Draw(painter *core.Painter) {
	if len(tb.Tabs) == 0 {
		return
	}

	s := tb.resolveColors()
	focused := tb.IsFocused()
	x := tb.Rect.X
	y := tb.Rect.Y
	maxX := tb.Rect.X + tb.Rect.W

	tm := theme.Get()
	activeStyle := tcell.StyleDefault.Foreground(s.ActiveFG).Background(s.ActiveBG)
	// Inactive tabs: normal text when bar unfocused, accent FG when bar focused
	inactiveFG := tm.GetSemanticColor("text.primary")
	if focused {
		inactiveFG = s.ActiveBG // accent color
	}
	inactiveStyle := tcell.StyleDefault.Foreground(inactiveFG).Background(s.InactiveBG)
	barStyle := tcell.StyleDefault.Foreground(s.BarBG).Background(s.BarBG)

	// Row 0: powerline tab row
	// Leading left triangle: FG = first tab's BG, BG = barBG
	firstBG := s.InactiveBG
	if tb.ActiveIdx == 0 {
		firstBG = s.ActiveBG
	}
	if x < maxX {
		painter.SetCell(x, y, plLeftTriangle, tcell.StyleDefault.Foreground(firstBG).Background(s.BarBG))
		x++
	}

	for i, tab := range tb.Tabs {
		tabLabel := " " + tab.Label + " "
		isActive := i == tb.ActiveIdx
		isHover := i == tb.hoverIdx && !isActive

		tabStyle := inactiveStyle
		if isActive {
			tabStyle = activeStyle
			if focused {
				tabStyle = tabStyle.Bold(true)
			}
		} else if isHover {
			tabStyle = inactiveStyle.Reverse(true)
		}

		// Draw tab label characters
		for _, ch := range tabLabel {
			if x >= maxX {
				break
			}
			painter.SetCell(x, y, ch, tabStyle)
			x++
		}

		// Draw separator between tabs (not after the last tab)
		if i < len(tb.Tabs)-1 && x < maxX {
			nextActive := (i + 1) == tb.ActiveIdx
			if isActive {
				// Leaving active tab: right triangle, FG=active, BG=next tab's BG
				nextBG := s.InactiveBG
				painter.SetCell(x, y, plRightTriangle, tcell.StyleDefault.Foreground(s.ActiveBG).Background(nextBG))
			} else if nextActive {
				// Entering active tab: left triangle, FG=active, BG=current tab's BG
				painter.SetCell(x, y, plLeftTriangle, tcell.StyleDefault.Foreground(s.ActiveBG).Background(s.InactiveBG))
			} else {
				// Between two inactive tabs: thin line separator
				painter.SetCell(x, y, plRightThinLine, tcell.StyleDefault.Foreground(s.BarBG).Background(s.InactiveBG))
			}
			x++
		}
	}

	// Trailing right triangle after last tab: FG = last tab's BG, BG = barBG
	if x < maxX {
		lastBG := s.InactiveBG
		if tb.ActiveIdx == len(tb.Tabs)-1 {
			lastBG = s.ActiveBG
		}
		painter.SetCell(x, y, plRightTriangle, tcell.StyleDefault.Foreground(lastBG).Background(s.BarBG))
		x++
	}

	// Fill rest of row 0 with bar background
	for x < maxX {
		painter.SetCell(x, y, ' ', barStyle)
		x++
	}

	// Row 1: blend row (if enabled)
	if !tb.Style.NoBlendRow && tb.Rect.H >= 2 {
		blendStyle := tcell.StyleDefault.Foreground(s.ActiveBG).Background(s.ContentBG)
		for bx := tb.Rect.X; bx < maxX; bx++ {
			painter.SetCell(bx, y+1, blendChar, blendStyle)
		}
	}
}

// HandleKey processes keyboard input for tab navigation.
// Returns true if the key was handled.
func (tb *TabBar) HandleKey(ev *tcell.EventKey) bool {
	if len(tb.Tabs) == 0 {
		return false
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
	tabIdx := tb.tabAtX(x)

	// Update hover state (only for non-active tabs)
	if tabIdx != tb.hoverIdx {
		tb.hoverIdx = tabIdx
		tb.invalidate()
	}

	// Handle click for tab selection
	if ev.Buttons() == tcell.Button1 {
		if tabIdx >= 0 && tabIdx != tb.ActiveIdx {
			tb.SetActive(tabIdx)
		}
		return true
	}

	return true
}

// tabAtX returns the tab index at the given x position, or -1 if none.
// Layout: [leftTri][" Label "][sep][" Label "][sep]...[rightTri][barBG...]
func (tb *TabBar) tabAtX(x int) int {
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

// GetKeyHints implements KeyHintsProvider from core package.
func (tb *TabBar) GetKeyHints() []core.KeyHint {
	return []core.KeyHint{
		{Key: "←→", Label: "Switch"},
		{Key: "1-9", Label: "Jump"},
	}
}
