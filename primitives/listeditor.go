// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: texelui/primitives/listeditor.go
// Summary: Generic list editor widget for arrays of map-based objects.

package primitives

import (
	"fmt"
	"sort"

	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

// ListEditorConfig configures the keys used by a ListEditor.
type ListEditorConfig struct {
	LabelKey  string // map key used as the row label (e.g. "id")
	ToggleKey string // map key for the checkbox (e.g. "enabled"); "" = no checkbox
}

// ListEditor is a widget that renders and edits a slice of map[string]interface{}
// objects. Each item has an optional checkbox, a label, expandable detail rows,
// and action buttons (move up/down, remove). An "Add" button appears at the bottom.
type ListEditor struct {
	core.BaseWidget
	config      ListEditorConfig
	items       []map[string]interface{}
	selectedIdx int
	expandedIdx int // -1 = none expanded
	OnChange    func([]map[string]interface{})
	inv         func(core.Rect)
}

// NewListEditor creates a new ListEditor with the given config.
func NewListEditor(cfg ListEditorConfig) *ListEditor {
	le := &ListEditor{
		config:      cfg,
		items:       []map[string]interface{}{},
		selectedIdx: 0,
		expandedIdx: -1,
	}
	le.SetFocusable(true)
	return le
}

// SetInvalidator allows the UI manager to inject a dirty-region invalidator.
func (le *ListEditor) SetInvalidator(fn func(core.Rect)) {
	le.inv = fn
}

// SetItems replaces the current items and resets selection.
func (le *ListEditor) SetItems(items []map[string]interface{}) {
	le.items = items
	le.selectedIdx = 0
	le.expandedIdx = -1
	le.invalidate()
}

// Items returns the current items slice.
func (le *ListEditor) Items() []map[string]interface{} {
	return le.items
}

// itemLabel returns the display label for the item.
func (le *ListEditor) itemLabel(item map[string]interface{}) string {
	if le.config.LabelKey == "" {
		return "(unnamed)"
	}
	if v, ok := item[le.config.LabelKey]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return "(unnamed)"
}

// itemToggle returns the boolean value of the ToggleKey for the item.
func (le *ListEditor) itemToggle(item map[string]interface{}) bool {
	if le.config.ToggleKey == "" {
		return false
	}
	if v, ok := item[le.config.ToggleKey]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// detailKeys returns the sorted list of keys excluding LabelKey and ToggleKey.
func (le *ListEditor) detailKeys(item map[string]interface{}) []string {
	keys := make([]string, 0, len(item))
	for k := range item {
		if k == le.config.LabelKey || k == le.config.ToggleKey {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// itemHeight returns the number of rows this item occupies.
func (le *ListEditor) itemHeight(idx int) int {
	if idx < 0 || idx >= len(le.items) {
		return 1
	}
	if idx == le.expandedIdx {
		return 1 + len(le.detailKeys(le.items[idx]))
	}
	return 1
}

// totalHeight returns the total number of rows needed for all items plus the add button.
func (le *ListEditor) totalHeight() int {
	total := 0
	for i := range le.items {
		total += le.itemHeight(i)
	}
	return total + 1 // +1 for the Add button row
}

// invalidate marks the widget dirty.
func (le *ListEditor) invalidate() {
	if le.inv != nil {
		le.inv(le.Rect)
	}
}

// fireChange calls OnChange (if set) and invalidates.
func (le *ListEditor) fireChange() {
	if le.OnChange != nil {
		le.OnChange(le.items)
	}
	le.invalidate()
}

// toggleItem flips the ToggleKey boolean for item at idx.
func (le *ListEditor) toggleItem(idx int) {
	if idx < 0 || idx >= len(le.items) || le.config.ToggleKey == "" {
		return
	}
	cur := le.itemToggle(le.items[idx])
	le.items[idx][le.config.ToggleKey] = !cur
	le.fireChange()
}

// moveUp swaps item at idx with idx-1.
func (le *ListEditor) moveUp(idx int) {
	if idx <= 0 || idx >= len(le.items) {
		return
	}
	le.items[idx-1], le.items[idx] = le.items[idx], le.items[idx-1]
	// Keep selection and expansion tracking consistent.
	if le.selectedIdx == idx {
		le.selectedIdx = idx - 1
	} else if le.selectedIdx == idx-1 {
		le.selectedIdx = idx
	}
	if le.expandedIdx == idx {
		le.expandedIdx = idx - 1
	} else if le.expandedIdx == idx-1 {
		le.expandedIdx = idx
	}
	le.fireChange()
}

// moveDown swaps item at idx with idx+1.
func (le *ListEditor) moveDown(idx int) {
	if idx < 0 || idx >= len(le.items)-1 {
		return
	}
	le.items[idx], le.items[idx+1] = le.items[idx+1], le.items[idx]
	if le.selectedIdx == idx {
		le.selectedIdx = idx + 1
	} else if le.selectedIdx == idx+1 {
		le.selectedIdx = idx
	}
	if le.expandedIdx == idx {
		le.expandedIdx = idx + 1
	} else if le.expandedIdx == idx+1 {
		le.expandedIdx = idx
	}
	le.fireChange()
}

// removeItem deletes the item at idx.
func (le *ListEditor) removeItem(idx int) {
	if idx < 0 || idx >= len(le.items) {
		return
	}
	le.items = append(le.items[:idx], le.items[idx+1:]...)
	// Clamp selectedIdx.
	if le.selectedIdx >= len(le.items) {
		le.selectedIdx = len(le.items) - 1
	}
	if le.selectedIdx < 0 {
		le.selectedIdx = 0
	}
	// Fix expandedIdx.
	if le.expandedIdx == idx {
		le.expandedIdx = -1
	} else if le.expandedIdx > idx {
		le.expandedIdx--
	}
	le.fireChange()
}

// addItem appends a new blank item, selects and expands it.
func (le *ListEditor) addItem() {
	item := map[string]interface{}{}
	if le.config.LabelKey != "" {
		item[le.config.LabelKey] = ""
	}
	if le.config.ToggleKey != "" {
		item[le.config.ToggleKey] = true
	}
	le.items = append(le.items, item)
	le.selectedIdx = len(le.items) - 1
	le.expandedIdx = le.selectedIdx
	le.fireChange()
}

// toggleExpand expands/collapses item at idx (does not fire OnChange).
func (le *ListEditor) toggleExpand(idx int) {
	if idx < 0 || idx >= len(le.items) {
		return
	}
	if le.expandedIdx == idx {
		le.expandedIdx = -1
	} else {
		le.expandedIdx = idx
	}
	le.invalidate()
}

// setDetailValue updates a detail field on item at itemIdx.
func (le *ListEditor) setDetailValue(itemIdx int, key string, value interface{}) {
	if itemIdx < 0 || itemIdx >= len(le.items) {
		return
	}
	le.items[itemIdx][key] = value
	le.fireChange()
}

// Draw renders the list editor into the painter.
func (le *ListEditor) Draw(p *core.Painter) {
	tm := theme.Get()
	fgPrimary := tm.GetSemanticColor("text.primary")
	fgMuted := tm.GetSemanticColor("text.muted")
	bgSurface := tm.GetSemanticColor("bg.surface")
	bgElevated := tm.GetSemanticColor("bg.elevated")
	accent := tm.GetSemanticColor("accent.primary")

	baseStyle := tcell.StyleDefault.Foreground(fgPrimary).Background(bgSurface)
	selectedStyle := tcell.StyleDefault.Foreground(fgPrimary).Background(bgElevated)
	accentStyle := tcell.StyleDefault.Foreground(accent).Background(bgSurface)
	mutedStyle := tcell.StyleDefault.Foreground(fgMuted).Background(bgSurface)

	// Fill background.
	p.Fill(le.Rect, ' ', baseStyle)

	x0 := le.Rect.X
	maxW := le.Rect.W
	maxY := le.Rect.Y + le.Rect.H

	row := le.Rect.Y

	for i, item := range le.items {
		if row >= maxY {
			break
		}

		selected := i == le.selectedIdx
		expanded := i == le.expandedIdx
		isFirst := i == 0
		isLast := i == len(le.items)-1

		// Choose row background.
		rowBG := bgSurface
		if selected {
			rowBG = bgElevated
		}
		hdrStyle := tcell.StyleDefault.Foreground(fgPrimary).Background(rowBG)
		hdrAccent := tcell.StyleDefault.Foreground(accent).Background(rowBG)
		hdrMuted := tcell.StyleDefault.Foreground(fgMuted).Background(rowBG)

		// Fill header row background.
		hdrRect := core.Rect{X: x0, Y: row, W: maxW, H: 1}
		p.Fill(hdrRect, ' ', hdrStyle)

		col := x0

		// Checkbox (if ToggleKey set).
		if le.config.ToggleKey != "" {
			ch := '☐'
			if le.itemToggle(item) {
				ch = '☑'
			}
			p.SetCell(col, row, ch, hdrAccent)
			col++
			p.SetCell(col, row, ' ', hdrStyle)
			col++
		}

		// Label.
		label := le.itemLabel(item)
		labelStart := col
		labelEnd := x0 + maxW - le.rightButtonsWidth(isFirst, isLast) - 1
		if labelEnd > labelStart {
			availW := labelEnd - labelStart
			if len([]rune(label)) > availW {
				label = string([]rune(label)[:availW])
			}
			p.DrawText(col, row, label, hdrStyle)
			col += len([]rune(label))
		}
		_ = col // silence unused warning; right-side buttons use absolute positions

		// Right-side buttons (drawn right-to-left from right edge).
		// Order right-to-left: expand(▸/▾), remove(✕), down(▼), up(▲).
		// Each button occupies 2 columns: [icon][space].
		rightCol := x0 + maxW - 1

		// Expand toggle: always present.
		expandCh := rune('▸')
		if expanded {
			expandCh = '▾'
		}
		p.SetCell(rightCol, row, expandCh, hdrAccent)
		rightCol--
		p.SetCell(rightCol, row, ' ', hdrStyle)
		rightCol--

		// Remove button.
		p.SetCell(rightCol, row, '✕', hdrAccent)
		rightCol--
		p.SetCell(rightCol, row, ' ', hdrStyle)
		rightCol--

		// Down button (hidden for last item).
		if !isLast {
			p.SetCell(rightCol, row, '▼', hdrAccent)
		} else {
			p.SetCell(rightCol, row, ' ', hdrMuted)
		}
		rightCol--
		p.SetCell(rightCol, row, ' ', hdrStyle)
		rightCol--

		// Up button (hidden for first item).
		if !isFirst {
			p.SetCell(rightCol, row, '▲', hdrAccent)
		} else {
			p.SetCell(rightCol, row, ' ', hdrMuted)
		}

		row++

		// Detail rows (only when expanded).
		if expanded {
			keys := le.detailKeys(item)
			for _, k := range keys {
				if row >= maxY {
					break
				}
				detailRect := core.Rect{X: x0, Y: row, W: maxW, H: 1}
				p.Fill(detailRect, ' ', baseStyle)
				valStr := fmt.Sprintf("%v", item[k])
				line := fmt.Sprintf("  %s: %s", k, valStr)
				// Draw key part muted, value part primary.
				keyPart := fmt.Sprintf("  %s: ", k)
				p.DrawText(x0, row, keyPart, mutedStyle)
				p.DrawText(x0+len([]rune(keyPart)), row, valStr, selectedStyle)
				_ = line
				row++
			}
		}
	}

	// Add button row.
	if row < maxY {
		addRect := core.Rect{X: x0, Y: row, W: maxW, H: 1}
		p.Fill(addRect, ' ', baseStyle)
		p.DrawText(x0, row, "[+ Add]", accentStyle)
	}
}

// rightButtonsWidth returns the number of columns used by right-side action buttons.
// Layout (right-to-left): expand(2), remove(2), down(2), up(2) = 8 total.
func (le *ListEditor) rightButtonsWidth(isFirst, isLast bool) int {
	// All four button slots are always reserved for alignment consistency.
	_ = isFirst
	_ = isLast
	return 8
}

// rowToItem maps an absolute Y coordinate to (itemIdx, isHeader, detailKeyIdx).
// Returns itemIdx=-1 if the row is the Add button or out of bounds.
func (le *ListEditor) rowToItem(absY int) (itemIdx int, isHeader bool, detailKeyIdx int) {
	row := le.Rect.Y
	for i := range le.items {
		if absY == row {
			return i, true, -1
		}
		row++
		if i == le.expandedIdx {
			keys := le.detailKeys(le.items[i])
			for ki := range keys {
				if absY == row {
					return i, false, ki
				}
				row++
			}
		}
	}
	return -1, false, -1
}

// HandleKey processes keyboard input.
func (le *ListEditor) HandleKey(ev *tcell.EventKey) bool {
	n := len(le.items)
	ctrl := ev.Modifiers()&tcell.ModCtrl != 0

	switch ev.Key() {
	case tcell.KeyUp:
		if ctrl {
			if n > 0 {
				le.moveUp(le.selectedIdx)
			}
		} else if le.selectedIdx > 0 {
			le.selectedIdx--
			le.invalidate()
		}
		return true

	case tcell.KeyDown:
		if ctrl {
			if n > 0 {
				le.moveDown(le.selectedIdx)
			}
		} else if le.selectedIdx < n-1 {
			le.selectedIdx++
			le.invalidate()
		}
		return true

	case tcell.KeyEnter:
		if n > 0 {
			le.toggleExpand(le.selectedIdx)
		}
		return true

	case tcell.KeyDelete:
		if n > 0 {
			le.removeItem(le.selectedIdx)
		}
		return true

	case tcell.KeyRune:
		switch ev.Rune() {
		case ' ':
			if n > 0 {
				le.toggleItem(le.selectedIdx)
			}
			return true
		case '+':
			le.addItem()
			return true
		case '-':
			if n > 0 {
				le.removeItem(le.selectedIdx)
			}
			return true
		}
	}

	return false
}

// HandleMouse processes mouse clicks on items and action buttons.
func (le *ListEditor) HandleMouse(ev *tcell.EventMouse) bool {
	if ev.Buttons() != tcell.Button1 {
		return false
	}

	mx, my := ev.Position()
	if !le.HitTest(mx, my) {
		return false
	}

	// Check if click is on the Add button row.
	addRow := le.Rect.Y
	for i := range le.items {
		addRow += le.itemHeight(i)
	}
	if my == addRow {
		le.addItem()
		return true
	}

	itemIdx, isHeader, _ := le.rowToItem(my)
	if itemIdx < 0 {
		return false
	}

	// Update selection.
	le.selectedIdx = itemIdx

	if !isHeader {
		// Click on a detail row — just update selection.
		le.invalidate()
		return true
	}

	// Determine which zone was clicked on the header row.
	x0 := le.Rect.X
	maxW := le.Rect.W
	isFirst := itemIdx == 0
	isLast := itemIdx == len(le.items)-1

	// Right-side buttons (absolute columns from right edge).
	// rightButtonsWidth = 8, columns assigned right-to-left:
	//   expand:  [maxW-1]
	//   (space): [maxW-2]
	//   remove:  [maxW-3]
	//   (space): [maxW-4]
	//   down:    [maxW-5]
	//   (space): [maxW-6]
	//   up:      [maxW-7]
	//   (space): [maxW-8]
	relX := mx - x0
	bw := le.rightButtonsWidth(isFirst, isLast)
	rightStart := maxW - bw // relative column where buttons begin

	if relX >= rightStart {
		// Which button?
		btnRel := relX - rightStart
		// Slots (each 2 wide): up(0-1), down(2-3), remove(4-5), expand(6-7)
		switch {
		case btnRel == 0 && !isFirst: // up arrow col
			le.moveUp(itemIdx)
		case btnRel == 2 && !isLast: // down arrow col
			le.moveDown(itemIdx)
		case btnRel == 4: // remove col
			le.removeItem(itemIdx)
		case btnRel == 6: // expand col
			le.toggleExpand(itemIdx)
		default:
			le.invalidate()
		}
		return true
	}

	// Checkbox zone (left edge, cols 0-1 when ToggleKey set).
	if le.config.ToggleKey != "" && relX < 2 {
		le.toggleItem(itemIdx)
		return true
	}

	// Click on label area — just toggle expand.
	le.toggleExpand(itemIdx)
	return true
}

// GetKeyHints implements core.KeyHintsProvider.
func (le *ListEditor) GetKeyHints() []core.KeyHint {
	return []core.KeyHint{
		{Key: "↑↓", Label: "Select"},
		{Key: "Enter", Label: "Expand"},
		{Key: "Space", Label: "Toggle"},
		{Key: "+/-", Label: "Add/Remove"},
		{Key: "^↑↓", Label: "Reorder"},
	}
}
