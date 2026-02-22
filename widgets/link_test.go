package widgets

import (
	"testing"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

func TestLink_Render(t *testing.T) {
	link := NewLink("Click here")
	link.SetPosition(0, 0)

	buf := createTestBuffer(20, 1)
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 20, H: 1})
	link.Draw(painter)

	// Verify the text content is rendered
	expected := "Click here"
	for i, ch := range expected {
		if i >= 20 {
			break
		}
		if buf[0][i].Ch != ch {
			t.Errorf("cell[0][%d].Ch = %q, want %q", i, buf[0][i].Ch, ch)
		}
	}

	// Verify that the style includes underline
	for i := range expected {
		if i >= 20 {
			break
		}
		_, _, attrs := buf[0][i].Style.Decompose()
		if attrs&tcell.AttrUnderline == 0 {
			t.Errorf("cell[0][%d] expected underline attribute, got attrs=%v", i, attrs)
		}
	}
}

func TestLink_RenderFocused(t *testing.T) {
	link := NewLink("Focused link")
	link.SetPosition(0, 0)
	link.Focus()

	buf := createTestBuffer(20, 1)
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 20, H: 1})
	link.Draw(painter)

	// Verify text is rendered
	expected := "Focused link"
	for i, ch := range expected {
		if i >= 20 {
			break
		}
		if buf[0][i].Ch != ch {
			t.Errorf("cell[0][%d].Ch = %q, want %q", i, buf[0][i].Ch, ch)
		}
	}

	// Verify underline is still applied even when focused
	for i := range expected {
		if i >= 20 {
			break
		}
		_, _, attrs := buf[0][i].Style.Decompose()
		if attrs&tcell.AttrUnderline == 0 {
			t.Errorf("focused cell[0][%d] expected underline attribute, got attrs=%v", i, attrs)
		}
	}
}

func TestLink_HandleKey(t *testing.T) {
	clicked := false
	link := NewLink("Test")
	link.OnClick = func() {
		clicked = true
	}

	// Enter key should trigger OnClick
	evEnter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	handled := link.HandleKey(evEnter)

	if !handled {
		t.Error("expected HandleKey to return true for Enter")
	}
	if !clicked {
		t.Error("expected OnClick to be triggered on Enter")
	}

	// Other keys should not trigger OnClick
	clicked = false

	evSpace := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	handled = link.HandleKey(evSpace)
	if handled {
		t.Error("expected HandleKey to return false for Space")
	}
	if clicked {
		t.Error("expected OnClick NOT to be triggered on Space")
	}

	evTab := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	handled = link.HandleKey(evTab)
	if handled {
		t.Error("expected HandleKey to return false for Tab")
	}
	if clicked {
		t.Error("expected OnClick NOT to be triggered on Tab")
	}

	evRune := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	handled = link.HandleKey(evRune)
	if handled {
		t.Error("expected HandleKey to return false for rune 'a'")
	}
	if clicked {
		t.Error("expected OnClick NOT to be triggered on rune 'a'")
	}
}

func TestLink_HandleKeyNoCallback(t *testing.T) {
	// OnClick is nil — Enter should not panic
	link := NewLink("Test")

	evEnter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	handled := link.HandleKey(evEnter)

	if !handled {
		t.Error("expected HandleKey to return true for Enter even without OnClick")
	}
}

func TestLink_Size(t *testing.T) {
	link := NewLink("Hello World")
	w, h := link.Size()

	if w != len("Hello World") {
		t.Errorf("expected width %d, got %d", len("Hello World"), w)
	}
	if h != 1 {
		t.Errorf("expected height 1, got %d", h)
	}
}

func TestLink_Focusable(t *testing.T) {
	link := NewLink("Test")
	if !link.Focusable() {
		t.Error("expected link to be focusable")
	}
}
