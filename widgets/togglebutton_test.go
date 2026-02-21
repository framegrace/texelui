package widgets

import (
	"testing"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

func TestToggleButtonDraw(t *testing.T) {
	tb := NewToggleButton("TFM")
	tb.SetPosition(2, 1)
	tb.Resize(3, 1)

	// --- Inactive: normal style (FG on BG) ---
	buf := createTestBuffer(10, 3)
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 10, H: 3})
	tb.Draw(painter)

	// Verify the label characters are at positions (2,1), (3,1), (4,1)
	expected := []rune{'T', 'F', 'M'}
	for i, ch := range expected {
		if buf[1][2+i].Ch != ch {
			t.Errorf("inactive draw: cell[1][%d].Ch = %q, want %q", 2+i, buf[1][2+i].Ch, ch)
		}
	}

	// Check inactive style is NOT reversed: FG and BG should be the normal style
	inactiveStyle := buf[1][2].Style
	inFG, inBG, _ := inactiveStyle.Decompose()

	// --- Active: reversed style (BG becomes FG, FG becomes BG) ---
	tb.Active = true
	buf = createTestBuffer(10, 3)
	painter = core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 10, H: 3})
	tb.Draw(painter)

	for i, ch := range expected {
		if buf[1][2+i].Ch != ch {
			t.Errorf("active draw: cell[1][%d].Ch = %q, want %q", 2+i, buf[1][2+i].Ch, ch)
		}
	}

	activeStyle := buf[1][2].Style
	actFG, actBG, _ := activeStyle.Decompose()

	// Active style should have FG/BG swapped compared to inactive
	if actFG != inBG {
		t.Errorf("active FG = %v, want inactive BG %v (swapped)", actFG, inBG)
	}
	if actBG != inFG {
		t.Errorf("active BG = %v, want inactive FG %v (swapped)", actBG, inFG)
	}
}

func TestToggleButtonSize(t *testing.T) {
	tests := []struct {
		label     string
		wantWidth int
	}{
		{"TFM", 3},
		{"WRP", 3},
		{"INS", 3},
		{"A", 1},
		{"SCROLL", 6},
	}

	for _, tt := range tests {
		tb := NewToggleButton(tt.label)
		w, h := tb.Size()
		if w != tt.wantWidth {
			t.Errorf("NewToggleButton(%q).Size() width = %d, want %d", tt.label, w, tt.wantWidth)
		}
		if h != 1 {
			t.Errorf("NewToggleButton(%q).Size() height = %d, want 1", tt.label, h)
		}
	}
}

func TestToggleButtonClick(t *testing.T) {
	tb := NewToggleButton("TFM")
	tb.SetPosition(0, 0)
	tb.Resize(3, 1)

	var callbackFired bool
	var callbackValue bool
	tb.OnToggle = func(active bool) {
		callbackFired = true
		callbackValue = active
	}

	if tb.Active {
		t.Fatal("expected Active=false initially")
	}

	// Click inside bounds
	ev := tcell.NewEventMouse(1, 0, tcell.Button1, tcell.ModNone)
	handled := tb.HandleMouse(ev)

	if !handled {
		t.Error("expected HandleMouse to return true for click inside bounds")
	}
	if !tb.Active {
		t.Error("expected Active=true after click")
	}
	if !callbackFired {
		t.Error("expected OnToggle callback to fire")
	}
	if !callbackValue {
		t.Error("expected OnToggle callback to receive true")
	}

	// Click again to toggle off
	callbackFired = false
	ev = tcell.NewEventMouse(2, 0, tcell.Button1, tcell.ModNone)
	handled = tb.HandleMouse(ev)

	if !handled {
		t.Error("expected HandleMouse to return true for second click")
	}
	if tb.Active {
		t.Error("expected Active=false after second click")
	}
	if !callbackFired {
		t.Error("expected OnToggle callback to fire on second click")
	}
	if callbackValue {
		t.Error("expected OnToggle callback to receive false")
	}
}

func TestToggleButtonClickOutside(t *testing.T) {
	tb := NewToggleButton("TFM")
	tb.SetPosition(0, 0)
	tb.Resize(3, 1)

	var callbackFired bool
	tb.OnToggle = func(active bool) {
		callbackFired = true
	}

	// Click outside bounds (x=5, which is beyond width=3)
	ev := tcell.NewEventMouse(5, 0, tcell.Button1, tcell.ModNone)
	handled := tb.HandleMouse(ev)

	if handled {
		t.Error("expected HandleMouse to return false for click outside bounds")
	}
	if tb.Active {
		t.Error("expected Active to remain false")
	}
	if callbackFired {
		t.Error("expected OnToggle callback NOT to fire")
	}
}

func TestToggleButtonDisabledIgnoresClick(t *testing.T) {
	tb := NewToggleButton("INS")
	tb.SetPosition(0, 0)
	tb.Resize(3, 1)
	tb.Disabled = true

	var callbackFired bool
	tb.OnToggle = func(active bool) {
		callbackFired = true
	}

	ev := tcell.NewEventMouse(1, 0, tcell.Button1, tcell.ModNone)
	handled := tb.HandleMouse(ev)

	if handled {
		t.Error("expected HandleMouse to return false when disabled")
	}
	if tb.Active {
		t.Error("expected Active to remain false when disabled")
	}
	if callbackFired {
		t.Error("expected OnToggle NOT to fire when disabled")
	}
}

func TestToggleButtonDisabledFadedStyle(t *testing.T) {
	tb := NewToggleButton("TUI")
	tb.SetPosition(0, 0)
	tb.Resize(3, 1)

	// Draw normal
	buf1 := createTestBuffer(10, 1)
	p1 := core.NewPainter(buf1, core.Rect{X: 0, Y: 0, W: 10, H: 1})
	tb.Draw(p1)
	normalFG, normalBG, _ := buf1[0][0].Style.Decompose()

	// Draw disabled
	tb.Disabled = true
	buf2 := createTestBuffer(10, 1)
	p2 := core.NewPainter(buf2, core.Rect{X: 0, Y: 0, W: 10, H: 1})
	tb.Draw(p2)
	disabledFG, disabledBG, _ := buf2[0][0].Style.Decompose()

	// BG should be unchanged
	if disabledBG != normalBG {
		t.Errorf("disabled BG = %v, want %v (unchanged)", disabledBG, normalBG)
	}
	// FG should be faded (different from normal)
	if disabledFG == normalFG {
		t.Error("disabled FG should differ from normal FG (should be faded)")
	}
}

func TestToggleButtonNoCallback(t *testing.T) {
	tb := NewToggleButton("INS")
	tb.SetPosition(0, 0)
	tb.Resize(3, 1)

	// OnToggle is nil — clicking should not panic
	ev := tcell.NewEventMouse(1, 0, tcell.Button1, tcell.ModNone)
	handled := tb.HandleMouse(ev)

	if !handled {
		t.Error("expected HandleMouse to return true for click inside bounds")
	}
	if !tb.Active {
		t.Error("expected Active=true after click, even without callback")
	}
}
