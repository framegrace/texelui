package widgets_test

import (
	"testing"

	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/widgets"
)

// Compile-time check that *Border implements FocusCycler.
var _ core.FocusCycler = (*widgets.Border)(nil)

func TestBorder_CycleFocus_DelegatesToChild(t *testing.T) {
	// Build a VBox with two focusable buttons inside a Border.
	vbox := widgets.NewVBox()
	btn1 := widgets.NewButton("A")
	btn2 := widgets.NewButton("B")
	vbox.AddChild(btn1)
	vbox.AddChild(btn2)
	vbox.Resize(40, 10)

	border := widgets.NewBorder()
	border.SetChild(vbox)
	border.Resize(42, 12)

	// Focus the border (delegates to VBox -> first button).
	border.Focus()

	// Cycle forward — should move from btn1 to btn2.
	if !border.CycleFocus(true) {
		t.Error("CycleFocus(true) should succeed moving to second button")
	}

	// Cycle forward again — at boundary, should return false.
	if border.CycleFocus(true) {
		t.Error("CycleFocus(true) should return false at boundary")
	}

	// TrapsFocus should return false (border is not a root).
	if border.TrapsFocus() {
		t.Error("Border.TrapsFocus() should return false")
	}
}

func TestBorder_CycleFocus_NoChild(t *testing.T) {
	border := widgets.NewBorder()
	border.Resize(10, 5)

	// No child — should return false.
	if border.CycleFocus(true) {
		t.Error("CycleFocus should return false with no child")
	}
}

func TestBorder_CycleFocus_NonCyclerChild(t *testing.T) {
	// A Label child is not a FocusCycler — CycleFocus returns false.
	lbl := widgets.NewLabel("hello")
	border := widgets.NewBorder()
	border.SetChild(lbl)
	border.Resize(20, 3)

	if border.CycleFocus(true) {
		t.Error("CycleFocus should return false when child is not a FocusCycler")
	}
}
