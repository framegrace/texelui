package primitives

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/framegrace/texelui/core"
)

func TestTabBar_NewTabBar(t *testing.T) {
	tabs := []TabItem{
		{Label: "Tab1", ID: "1"},
		{Label: "Tab2", ID: "2"},
		{Label: "Tab3", ID: "3"},
	}

	tb := NewTabBar(5, 10, 40, tabs)

	if len(tb.Tabs) != 3 {
		t.Errorf("expected 3 tabs, got %d", len(tb.Tabs))
	}

	x, y := tb.Position()
	if x != 5 || y != 10 {
		t.Errorf("expected position (5,10), got (%d,%d)", x, y)
	}

	w, h := tb.Size()
	if w != 40 || h != 2 {
		t.Errorf("expected size (40,2), got (%d,%d)", w, h)
	}

	if tb.ActiveIdx != 0 {
		t.Errorf("expected initial ActiveIdx 0, got %d", tb.ActiveIdx)
	}

	if !tb.Focusable() {
		t.Error("expected TabBar to be focusable")
	}
}

func TestTabBar_TabBarHeight(t *testing.T) {
	tabs := []TabItem{{Label: "A"}}

	// Default: blend row enabled, height = 2
	tb := NewTabBar(0, 0, 20, tabs)
	if h := tb.TabBarHeight(); h != 2 {
		t.Errorf("expected default TabBarHeight 2, got %d", h)
	}
	_, sh := tb.Size()
	if sh != 2 {
		t.Errorf("expected Size height 2, got %d", sh)
	}

	// NoBlendRow: height = 1
	tb2 := &TabBar{
		Tabs:     tabs,
		Style:    TabBarStyle{NoBlendRow: true},
		hoverIdx: -1,
	}
	tb2.SetPosition(0, 0)
	tb2.Resize(20, tb2.TabBarHeight())
	if h := tb2.TabBarHeight(); h != 1 {
		t.Errorf("expected NoBlendRow TabBarHeight 1, got %d", h)
	}
	_, sh = tb2.Size()
	if sh != 1 {
		t.Errorf("expected NoBlendRow Size height 1, got %d", sh)
	}
}

func TestTabBar_SetActive(t *testing.T) {
	tabs := []TabItem{
		{Label: "A"},
		{Label: "B"},
		{Label: "C"},
	}
	tb := NewTabBar(0, 0, 30, tabs)

	callCount := 0
	lastIdx := -1
	tb.OnChange = func(idx int) {
		callCount++
		lastIdx = idx
	}

	// Set to valid index
	tb.SetActive(1)
	if tb.ActiveIdx != 1 {
		t.Errorf("expected ActiveIdx 1, got %d", tb.ActiveIdx)
	}
	if callCount != 1 || lastIdx != 1 {
		t.Errorf("expected OnChange called once with idx 1, got callCount=%d lastIdx=%d", callCount, lastIdx)
	}

	// Set to same index (should not trigger callback)
	tb.SetActive(1)
	if callCount != 1 {
		t.Errorf("expected no additional OnChange call, got callCount=%d", callCount)
	}

	// Set to invalid index (negative)
	tb.SetActive(-1)
	if tb.ActiveIdx != 1 {
		t.Errorf("expected ActiveIdx to remain 1 after invalid set, got %d", tb.ActiveIdx)
	}

	// Set to invalid index (too large)
	tb.SetActive(10)
	if tb.ActiveIdx != 1 {
		t.Errorf("expected ActiveIdx to remain 1 after invalid set, got %d", tb.ActiveIdx)
	}
}

func TestTabBar_ActiveTab(t *testing.T) {
	tabs := []TabItem{
		{Label: "First", ID: "first"},
		{Label: "Second", ID: "second"},
	}
	tb := NewTabBar(0, 0, 30, tabs)

	active := tb.ActiveTab()
	if active.Label != "First" || active.ID != "first" {
		t.Errorf("expected first tab, got %+v", active)
	}

	tb.SetActive(1)
	active = tb.ActiveTab()
	if active.Label != "Second" || active.ID != "second" {
		t.Errorf("expected second tab, got %+v", active)
	}
}

func TestTabBar_HandleKey_LeftRight(t *testing.T) {
	tabs := []TabItem{
		{Label: "A"},
		{Label: "B"},
		{Label: "C"},
	}
	tb := NewTabBar(0, 0, 30, tabs)

	// Right arrow
	ev := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	handled := tb.HandleKey(ev)
	if !handled || tb.ActiveIdx != 1 {
		t.Errorf("expected Right to move to idx 1, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}

	// Right again
	handled = tb.HandleKey(ev)
	if !handled || tb.ActiveIdx != 2 {
		t.Errorf("expected Right to move to idx 2, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}

	// Right at end (should not handle - at boundary)
	handled = tb.HandleKey(ev)
	if handled || tb.ActiveIdx != 2 {
		t.Errorf("expected Right at end to not handle, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}

	// Left arrow
	ev = tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	handled = tb.HandleKey(ev)
	if !handled || tb.ActiveIdx != 1 {
		t.Errorf("expected Left to move to idx 1, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}

	// Left to beginning
	handled = tb.HandleKey(ev)
	if !handled || tb.ActiveIdx != 0 {
		t.Errorf("expected Left to move to idx 0, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}

	// Left at beginning (should not handle - at boundary)
	handled = tb.HandleKey(ev)
	if handled || tb.ActiveIdx != 0 {
		t.Errorf("expected Left at beginning to not handle, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}
}

func TestTabBar_HandleKey_HomeEnd(t *testing.T) {
	tabs := []TabItem{
		{Label: "A"},
		{Label: "B"},
		{Label: "C"},
		{Label: "D"},
	}
	tb := NewTabBar(0, 0, 40, tabs)
	tb.SetActive(2) // Start in middle

	// End key
	ev := tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
	handled := tb.HandleKey(ev)
	if !handled || tb.ActiveIdx != 3 {
		t.Errorf("expected End to go to last tab, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}

	// End when already at end (should not handle)
	handled = tb.HandleKey(ev)
	if handled {
		t.Error("expected End at end to not handle")
	}

	// Home key
	ev = tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
	handled = tb.HandleKey(ev)
	if !handled || tb.ActiveIdx != 0 {
		t.Errorf("expected Home to go to first tab, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}

	// Home when already at beginning (should not handle)
	handled = tb.HandleKey(ev)
	if handled {
		t.Error("expected Home at beginning to not handle")
	}
}

func TestTabBar_HandleKey_NumberKeys(t *testing.T) {
	tabs := []TabItem{
		{Label: "One"},
		{Label: "Two"},
		{Label: "Three"},
	}
	tb := NewTabBar(0, 0, 40, tabs)

	// Press '2' to select second tab
	ev := tcell.NewEventKey(tcell.KeyRune, '2', tcell.ModNone)
	handled := tb.HandleKey(ev)
	if !handled || tb.ActiveIdx != 1 {
		t.Errorf("expected '2' to select idx 1, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}

	// Press '1' to select first tab
	ev = tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModNone)
	handled = tb.HandleKey(ev)
	if !handled || tb.ActiveIdx != 0 {
		t.Errorf("expected '1' to select idx 0, got handled=%v idx=%d", handled, tb.ActiveIdx)
	}

	// Press '9' (out of range - should not handle)
	ev = tcell.NewEventKey(tcell.KeyRune, '9', tcell.ModNone)
	handled = tb.HandleKey(ev)
	if handled {
		t.Error("expected '9' to not handle for 3-tab bar")
	}

	// Press same number as current (should not handle)
	ev = tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModNone)
	handled = tb.HandleKey(ev)
	if handled {
		t.Error("expected pressing current tab number to not handle")
	}
}

// makeBuf creates a rows x cols Cell buffer for testing Draw output.
func makeBuf(cols, rows int) [][]core.Cell {
	buf := make([][]core.Cell, rows)
	for r := range buf {
		buf[r] = make([]core.Cell, cols)
	}
	return buf
}

func TestTabBar_Draw_Powerline(t *testing.T) {
	tabs := []TabItem{
		{Label: "Alpha"},
		{Label: "Beta"},
	}
	tb := NewTabBar(0, 0, 40, tabs)
	// ActiveIdx defaults to 0 (Alpha is active)

	buf := makeBuf(40, 2)
	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 40, H: 2})
	tb.Draw(p)

	// Col 0: leading left triangle
	if buf[0][0].Ch != plLeftTriangle {
		t.Errorf("expected left triangle at (0,0), got %c (U+%04X)", buf[0][0].Ch, buf[0][0].Ch)
	}

	// Col 1-7: " Alpha " (7 chars)
	label := " Alpha "
	for i, ch := range label {
		if buf[0][1+i].Ch != ch {
			t.Errorf("expected %c at col %d, got %c", ch, 1+i, buf[0][1+i].Ch)
		}
	}

	// Col 8: separator (right triangle leaving active tab)
	if buf[0][8].Ch != plRightTriangle {
		t.Errorf("expected right triangle separator at col 8, got %c (U+%04X)", buf[0][8].Ch, buf[0][8].Ch)
	}

	// Col 9-14: " Beta " (6 chars)
	label2 := " Beta "
	for i, ch := range label2 {
		if buf[0][9+i].Ch != ch {
			t.Errorf("expected %c at col %d, got %c", ch, 9+i, buf[0][9+i].Ch)
		}
	}

	// Col 15: trailing right triangle
	if buf[0][15].Ch != plRightTriangle {
		t.Errorf("expected trailing right triangle at col 15, got %c (U+%04X)", buf[0][15].Ch, buf[0][15].Ch)
	}
}

func TestTabBar_Draw_BlendRow(t *testing.T) {
	tabs := []TabItem{{Label: "Tab"}}
	tb := NewTabBar(0, 0, 20, tabs)

	buf := makeBuf(20, 2)
	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 20, H: 2})
	tb.Draw(p)

	// Row 1 should be all blend chars across full width
	for x := 0; x < 20; x++ {
		if buf[1][x].Ch != blendChar {
			t.Errorf("expected blend char at (1,%d), got %c (U+%04X)", x, buf[1][x].Ch, buf[1][x].Ch)
		}
	}
}

func TestTabBar_Draw_SingleTab(t *testing.T) {
	tabs := []TabItem{{Label: "Solo"}}
	tb := NewTabBar(0, 0, 20, tabs)

	buf := makeBuf(20, 2)
	p := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 20, H: 2})
	tb.Draw(p)

	// Expected: leftTriangle + " Solo " + rightTriangle + bar fill
	if buf[0][0].Ch != plLeftTriangle {
		t.Errorf("expected left triangle at col 0, got %c", buf[0][0].Ch)
	}

	label := " Solo "
	for i, ch := range label {
		if buf[0][1+i].Ch != ch {
			t.Errorf("expected %c at col %d, got %c", ch, 1+i, buf[0][1+i].Ch)
		}
	}

	// Col 7: trailing right triangle
	if buf[0][7].Ch != plRightTriangle {
		t.Errorf("expected right triangle at col 7, got %c (U+%04X)", buf[0][7].Ch, buf[0][7].Ch)
	}

	// Rest should be bar fill (spaces)
	for x := 8; x < 20; x++ {
		if buf[0][x].Ch != ' ' {
			t.Errorf("expected space at col %d, got %c", x, buf[0][x].Ch)
		}
	}
}

func TestTabBar_EmptyTabs(t *testing.T) {
	tb := NewTabBar(0, 0, 30, []TabItem{})

	// HandleKey should return false for empty tabs
	ev := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	if tb.HandleKey(ev) {
		t.Error("expected HandleKey to return false for empty tabs")
	}

	// ActiveTab should return empty item
	active := tb.ActiveTab()
	if active.Label != "" || active.ID != "" {
		t.Errorf("expected empty tab, got %+v", active)
	}
}
