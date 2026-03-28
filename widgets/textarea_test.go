package widgets_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/framegrace/texelui/widgets"
)

func TestTextArea_EditMode(t *testing.T) {
	ta := widgets.NewTextArea()
	ta.SetPosition(0, 0)
	ta.Resize(20, 4)
	ta.SetText("hello\nworld")
	ta.Focus()

	// Enter activates edit mode first
	enterEv := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	if handled := ta.HandleKey(enterEv); !handled {
		t.Fatal("Enter should be consumed to activate editing")
	}

	// Up/Down should be consumed when editing
	downEv := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	if handled := ta.HandleKey(downEv); !handled {
		t.Error("Down should be consumed when editing")
	}

	upEv := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	if handled := ta.HandleKey(upEv); !handled {
		t.Error("Up should be consumed when editing")
	}

	// Escape deactivates editing
	escEv := tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone)
	if handled := ta.HandleKey(escEv); !handled {
		t.Fatal("Escape should be consumed to deactivate editing")
	}

	// Up/Down should NOT be consumed when not editing
	if handled := ta.HandleKey(downEv); handled {
		t.Error("Down should NOT be consumed when not editing")
	}
	if handled := ta.HandleKey(upEv); handled {
		t.Error("Up should NOT be consumed when not editing")
	}
}

func TestTextArea_UpDownPassThroughWhenNotEditing(t *testing.T) {
	ta := widgets.NewTextArea()
	ta.SetPosition(0, 0)
	ta.Resize(20, 4)
	ta.SetText("line one\nline two\nline three")
	ta.Focus()

	// Without entering edit mode, Up/Down should pass through
	downEv := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	if handled := ta.HandleKey(downEv); handled {
		t.Error("Down should pass through when not in edit mode")
	}

	upEv := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	if handled := ta.HandleKey(upEv); handled {
		t.Error("Up should pass through when not in edit mode")
	}

	// Left/Right should still be consumed (horizontal movement is always allowed)
	rightEv := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	if handled := ta.HandleKey(rightEv); !handled {
		t.Error("Right should still be consumed when not editing")
	}
}

func TestTextArea_BlurResetsEditMode(t *testing.T) {
	ta := widgets.NewTextArea()
	ta.SetPosition(0, 0)
	ta.Resize(20, 4)
	ta.SetText("hello\nworld")
	ta.Focus()

	// Activate editing with Enter
	enterEv := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	ta.HandleKey(enterEv)

	// Verify editing is active (Up should be consumed)
	upEv := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	if handled := ta.HandleKey(upEv); !handled {
		t.Fatal("Up should be consumed when editing is active")
	}

	// Blur resets editing
	ta.Blur()

	// Re-focus
	ta.Focus()

	// Up/Down should NOT be consumed (editing was reset by Blur)
	downEv := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	if handled := ta.HandleKey(downEv); handled {
		t.Error("Down should NOT be consumed after Blur reset editing")
	}
	if handled := ta.HandleKey(upEv); handled {
		t.Error("Up should NOT be consumed after Blur reset editing")
	}
}

func TestTextArea_EscapeWhenNotEditing(t *testing.T) {
	ta := widgets.NewTextArea()
	ta.SetPosition(0, 0)
	ta.Resize(20, 4)
	ta.Focus()

	// Escape when not editing should not be consumed
	escEv := tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone)
	if handled := ta.HandleKey(escEv); handled {
		t.Error("Escape should NOT be consumed when not in edit mode")
	}
}
