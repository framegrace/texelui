package widgets_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/framegrace/texelui/widgets"
)

func TestComboBox_UpDownDontOpenWhenClosed(t *testing.T) {
	items := []string{"Apple", "Banana", "Cherry"}
	cb := widgets.NewComboBox(items, false)
	cb.SetPosition(0, 0)
	cb.Focus()

	// Down should not be consumed when closed
	downEv := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	if handled := cb.HandleKey(downEv); handled {
		t.Error("Down should NOT be consumed when dropdown is closed")
	}

	// Up should not be consumed when closed
	upEv := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	if handled := cb.HandleKey(upEv); handled {
		t.Error("Up should NOT be consumed when dropdown is closed")
	}

	// Verify dropdown is still closed (IsModal returns false when not expanded)
	if cb.IsModal() {
		t.Error("ComboBox should not be expanded after Up/Down")
	}
}

func TestComboBox_EnterOpensDropdown(t *testing.T) {
	items := []string{"Apple", "Banana", "Cherry"}
	cb := widgets.NewComboBox(items, false)
	cb.SetPosition(0, 0)
	cb.Focus()

	// Verify not expanded initially
	if cb.IsModal() {
		t.Fatal("ComboBox should not be expanded initially")
	}

	// Enter should open the dropdown
	enterEv := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	if handled := cb.HandleKey(enterEv); !handled {
		t.Error("Enter should be consumed when opening dropdown")
	}

	// Verify expanded (IsModal returns true when expanded)
	if !cb.IsModal() {
		t.Error("ComboBox should be expanded after Enter")
	}
}

func TestComboBox_UpDownWorkWhenExpanded(t *testing.T) {
	items := []string{"Apple", "Banana", "Cherry"}
	cb := widgets.NewComboBox(items, false)
	cb.SetPosition(0, 0)
	cb.Resize(20, 1)
	cb.Focus()

	// Open dropdown with Enter
	enterEv := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	cb.HandleKey(enterEv)

	if !cb.IsModal() {
		t.Fatal("ComboBox should be expanded after Enter")
	}

	// Down should be consumed when expanded (navigates the list)
	downEv := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	if handled := cb.HandleKey(downEv); !handled {
		t.Error("Down should be consumed when dropdown is expanded")
	}

	// Up should be consumed when expanded
	upEv := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	if handled := cb.HandleKey(upEv); !handled {
		t.Error("Up should be consumed when dropdown is expanded")
	}
}

func TestComboBox_EscapeClosesDropdown(t *testing.T) {
	items := []string{"Apple", "Banana", "Cherry"}
	cb := widgets.NewComboBox(items, false)
	cb.SetPosition(0, 0)
	cb.Focus()

	// Open dropdown
	enterEv := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	cb.HandleKey(enterEv)

	if !cb.IsModal() {
		t.Fatal("ComboBox should be expanded")
	}

	// Escape should close it
	escEv := tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone)
	if handled := cb.HandleKey(escEv); !handled {
		t.Error("Escape should be consumed when closing dropdown")
	}

	if cb.IsModal() {
		t.Error("ComboBox should not be expanded after Escape")
	}
}
