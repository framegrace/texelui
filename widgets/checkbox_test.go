package widgets

import (
	"testing"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

func TestCheckbox_Toggle(t *testing.T) {
	changeCount := 0
	var lastState bool
	cb := NewCheckbox("Enable")
	cb.OnChange = func(checked bool) {
		changeCount++
		lastState = checked
	}

	if cb.Checked {
		t.Fatal("expected checkbox to be unchecked by default")
	}

	// Toggle with Space key
	ev := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	handled := cb.HandleKey(ev)

	if !handled {
		t.Error("expected HandleKey to return true for Space")
	}
	if !cb.Checked {
		t.Error("expected checkbox to be checked after Space toggle")
	}
	if changeCount != 1 {
		t.Errorf("expected OnChange called once, got %d", changeCount)
	}
	if !lastState {
		t.Error("expected OnChange to receive true")
	}

	// Toggle again with Space
	cb.HandleKey(ev)
	if cb.Checked {
		t.Error("expected checkbox to be unchecked after second toggle")
	}
	if changeCount != 2 {
		t.Errorf("expected OnChange called twice, got %d", changeCount)
	}
	if lastState {
		t.Error("expected OnChange to receive false")
	}

	// Toggle with Enter key
	evEnter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	cb.HandleKey(evEnter)
	if !cb.Checked {
		t.Error("expected checkbox to be checked after Enter toggle")
	}
	if changeCount != 3 {
		t.Errorf("expected OnChange called three times, got %d", changeCount)
	}

	// Other keys should not toggle
	evTab := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	handled = cb.HandleKey(evTab)
	if handled {
		t.Error("expected HandleKey to return false for Tab")
	}
	if changeCount != 3 {
		t.Errorf("expected OnChange still at 3 after unrelated key, got %d", changeCount)
	}
}

func TestCheckbox_Render(t *testing.T) {
	cb := NewCheckbox("Option A")
	cb.SetPosition(0, 0)

	// --- Unchecked rendering ---
	buf := createTestBuffer(20, 1)
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 20, H: 1})
	cb.Draw(painter)

	// Expect "[ ] Option A"
	uncheckedText := "[ ] Option A"
	for i, ch := range uncheckedText {
		if i >= 20 {
			break
		}
		if buf[0][i].Ch != ch {
			t.Errorf("unchecked: cell[0][%d].Ch = %q, want %q", i, buf[0][i].Ch, ch)
		}
	}

	// --- Checked rendering ---
	cb.Checked = true
	buf = createTestBuffer(20, 1)
	painter = core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 20, H: 1})
	cb.Draw(painter)

	// Expect "[x] Option A"
	checkedText := "[x] Option A"
	for i, ch := range checkedText {
		if i >= 20 {
			break
		}
		if buf[0][i].Ch != ch {
			t.Errorf("checked: cell[0][%d].Ch = %q, want %q", i, buf[0][i].Ch, ch)
		}
	}
}
