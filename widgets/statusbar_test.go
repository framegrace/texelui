package widgets

import (
	"testing"
	"time"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

// TestStatusBarShowMessageDuringHandleKey reproduces the freeze that occurs
// when StatusBar.ShowSuccess is called from within a widget's OnChange callback
// during HandleKey processing.
func TestStatusBarShowMessageDuringHandleKey(t *testing.T) {
	ui := core.NewUIManager()
	ui.Resize(80, 24)

	// Create and set up StatusBar
	sb := NewStatusBar()
	sb.SetPosition(0, 22)
	sb.Resize(80, 2)
	ui.SetStatusBar(sb)

	// Create a button that calls StatusBar.ShowSuccess when clicked
	btn := NewButton("Test")
	btn.SetPosition(10, 10)
	btn.OnClick = func() {
		// This simulates what happens when a widget's OnChange calls showSuccess
		sb.ShowSuccess("Success!")
	}

	ui.AddWidget(btn)
	ui.Focus(btn)

	// Set up a refresh notifier (like the runtime runner does)
	refreshCh := make(chan bool, 1)
	ui.SetRefreshNotifier(refreshCh)

	// Simulate pressing Enter on the button - this should trigger OnClick
	// which calls StatusBar.ShowSuccess
	done := make(chan struct{})
	go func() {
		ev := tcell.NewEventKey(tcell.KeyEnter, 0, 0)
		ui.HandleKey(ev)
		close(done)
	}()

	// Wait for completion with timeout
	select {
	case <-done:
		// Good - HandleKey completed without blocking
	case <-time.After(2 * time.Second):
		t.Fatal("HandleKey blocked - potential deadlock in StatusBar.ShowSuccess during HandleKey")
	}

	// Clean up
	sb.Stop()
}

// TestStatusBarShowMessageFromCallback tests calling ShowSuccess from a callback
// that's triggered during key handling, similar to how config editor uses it.
func TestStatusBarShowMessageFromCallback(t *testing.T) {
	ui := core.NewUIManager()
	ui.Resize(80, 24)

	// Create and set up StatusBar
	sb := NewStatusBar()
	sb.SetPosition(0, 22)
	sb.Resize(80, 2)
	ui.SetStatusBar(sb)

	// Create an input that calls StatusBar.ShowSuccess on change
	input := NewInput()
	input.SetPosition(10, 10)
	input.Resize(30, 1)
	input.OnChange = func(text string) {
		sb.ShowSuccess("Changed to: " + text)
	}

	ui.AddWidget(input)
	ui.Focus(input)

	// Set up a refresh notifier
	refreshCh := make(chan bool, 1)
	ui.SetRefreshNotifier(refreshCh)

	// Simulate typing a character - this should trigger OnChange
	done := make(chan struct{})
	go func() {
		ev := tcell.NewEventKey(tcell.KeyRune, 'a', 0)
		ui.HandleKey(ev)
		close(done)
	}()

	// Wait for completion with timeout
	select {
	case <-done:
		// Good - HandleKey completed
	case <-time.After(2 * time.Second):
		t.Fatal("HandleKey blocked - deadlock when calling StatusBar.ShowSuccess from OnChange")
	}

	// Verify the message was added
	if len(sb.messages) == 0 {
		t.Error("Expected message to be added to StatusBar")
	}

	// Clean up
	sb.Stop()
}

// TestStatusBarTickerDuringHandleKey tests that the StatusBar ticker doesn't
// cause issues when HandleKey is in progress.
func TestStatusBarTickerDuringHandleKey(t *testing.T) {
	ui := core.NewUIManager()
	ui.Resize(80, 24)

	// Create and set up StatusBar with a short message that will expire
	sb := NewStatusBar()
	sb.SetPosition(0, 22)
	sb.Resize(80, 2)
	sb.DefaultMessageDuration = 50 * time.Millisecond // Very short duration
	ui.SetStatusBar(sb)

	// Add a message that will expire soon
	sb.ShowMessage("Expiring soon")

	// Create a simple focusable widget
	btn := NewButton("Test")
	btn.SetPosition(10, 10)
	ui.AddWidget(btn)
	ui.Focus(btn)

	// Set up a refresh notifier
	refreshCh := make(chan bool, 1)
	ui.SetRefreshNotifier(refreshCh)

	// Drain refresh channel in background
	stopDrain := make(chan struct{})
	go func() {
		for {
			select {
			case <-refreshCh:
			case <-stopDrain:
				return
			}
		}
	}()

	// Simulate multiple key events while the ticker might be expiring messages
	for i := 0; i < 20; i++ {
		done := make(chan struct{})
		go func() {
			ev := tcell.NewEventKey(tcell.KeyRune, 'x', 0)
			ui.HandleKey(ev)
			close(done)
		}()

		select {
		case <-done:
			// Good
		case <-time.After(1 * time.Second):
			t.Fatalf("HandleKey blocked on iteration %d", i)
		}

		// Small delay to let ticker potentially fire
		time.Sleep(10 * time.Millisecond)
	}

	// Clean up - stop StatusBar first, then stop draining
	sb.Stop()
	close(stopDrain)
}

// TestStatusBarRenderDuringShowMessage tests that calling Render while
// ShowMessage is in progress doesn't cause a deadlock.
func TestStatusBarRenderDuringShowMessage(t *testing.T) {
	ui := core.NewUIManager()
	ui.Resize(80, 24)

	sb := NewStatusBar()
	sb.SetPosition(0, 22)
	sb.Resize(80, 2)
	ui.SetStatusBar(sb)

	btn := NewButton("Test")
	btn.SetPosition(10, 10)
	ui.AddWidget(btn)
	ui.Focus(btn)

	refreshCh := make(chan bool, 1)
	ui.SetRefreshNotifier(refreshCh)

	// Simulate the runtime runner pattern: render after each event
	for i := 0; i < 10; i++ {
		done := make(chan struct{})
		go func() {
			// Show a message (like config editor does)
			sb.ShowSuccess("Saved.")
			// Then render (like runtime runner does after HandleKey)
			ui.Render()
			close(done)
		}()

		select {
		case <-done:
			// Good
		case <-time.After(2 * time.Second):
			t.Fatalf("ShowSuccess + Render blocked on iteration %d", i)
		}
	}

	sb.Stop()
}

// TestStatusBarLeftWidgets verifies that SetLeftWidgets positions child widgets
// on the content row and that their labels appear at the correct positions.
func TestStatusBarLeftWidgets(t *testing.T) {
	sb := NewStatusBar()
	sb.SetPosition(0, 0)
	sb.Resize(40, 2)

	tb1 := NewToggleButton("TFM")
	tb2 := NewToggleButton("WRP")
	sb.SetLeftWidgets([]core.Widget{tb1, tb2})

	buf := createTestBuffer(40, 2)
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 40, H: 2})
	sb.Draw(painter)

	// Content is on row 1. First widget starts at x=1 (1-char left padding).
	// tb1 "TFM" at x=1,2,3 then 1-char gap, tb2 "WRP" at x=5,6,7
	for i, ch := range []rune{'T', 'F', 'M'} {
		if buf[1][1+i].Ch != ch {
			t.Errorf("tb1: cell[1][%d].Ch = %q, want %q", 1+i, buf[1][1+i].Ch, ch)
		}
	}
	for i, ch := range []rune{'W', 'R', 'P'} {
		if buf[1][5+i].Ch != ch {
			t.Errorf("tb2: cell[1][%d].Ch = %q, want %q", 5+i, buf[1][5+i].Ch, ch)
		}
	}
}

// TestStatusBarHandleMouse verifies that mouse clicks on left-side widgets
// are forwarded and that HandleMouse returns true when a widget handles it.
func TestStatusBarHandleMouse(t *testing.T) {
	sb := NewStatusBar()
	sb.SetPosition(0, 0)
	sb.Resize(40, 2)

	var toggled bool
	tb1 := NewToggleButton("TFM")
	tb1.OnToggle = func(active bool) {
		toggled = true
	}
	sb.SetLeftWidgets([]core.Widget{tb1})

	// Draw first to trigger layout (positions the widgets)
	buf := createTestBuffer(40, 2)
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 40, H: 2})
	sb.Draw(painter)

	// Click on tb1 at (1, 1) — content row, first widget position
	ev := tcell.NewEventMouse(1, 1, tcell.Button1, tcell.ModNone)
	handled := sb.HandleMouse(ev)

	if !handled {
		t.Error("expected HandleMouse to return true for click on toggle button")
	}
	if !toggled {
		t.Error("expected OnToggle callback to fire")
	}
	if !tb1.Active {
		t.Error("expected toggle button to be active after click")
	}
}

// TestStatusBarMouseOutside verifies that clicks outside the status bar
// rect are not handled.
func TestStatusBarMouseOutside(t *testing.T) {
	sb := NewStatusBar()
	sb.SetPosition(0, 10)
	sb.Resize(40, 2)

	tb1 := NewToggleButton("TFM")
	sb.SetLeftWidgets([]core.Widget{tb1})

	// Draw to trigger layout
	buf := createTestBuffer(40, 12)
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 40, H: 12})
	sb.Draw(painter)

	// Click outside the status bar (y=5 is above the bar at y=10..11)
	ev := tcell.NewEventMouse(1, 5, tcell.Button1, tcell.ModNone)
	handled := sb.HandleMouse(ev)

	if handled {
		t.Error("expected HandleMouse to return false for click outside status bar")
	}
}

// TestStatusBarLeftWidgetsOverridesHints verifies that when leftWidgets are set,
// key hints text does NOT appear (widgets take priority).
func TestStatusBarLeftWidgetsOverridesHints(t *testing.T) {
	sb := NewStatusBar()
	sb.SetPosition(0, 0)
	sb.Resize(60, 2)

	// Manually set leftText to simulate key hints being present
	sb.mu.Lock()
	sb.leftText = "Tab:Next S-Tab:Prev"
	sb.mu.Unlock()

	// Set left widgets — should override the key hints
	tb1 := NewToggleButton("TFM")
	sb.SetLeftWidgets([]core.Widget{tb1})

	buf := createTestBuffer(60, 2)
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 60, H: 2})
	sb.Draw(painter)

	// The toggle button label "TFM" should appear at x=1 on row 1
	for i, ch := range []rune{'T', 'F', 'M'} {
		if buf[1][1+i].Ch != ch {
			t.Errorf("tb1: cell[1][%d].Ch = %q, want %q", 1+i, buf[1][1+i].Ch, ch)
		}
	}

	// The key hints text should NOT appear. Check that "Tab" is not rendered
	// anywhere on row 1 past the widget area.
	hintText := []rune("Tab:Next")
	for x := 0; x <= 60-len(hintText); x++ {
		match := true
		for i, ch := range hintText {
			if x+i >= 60 || buf[1][x+i].Ch != ch {
				match = false
				break
			}
		}
		if match {
			t.Errorf("found key hints text 'Tab:Next' at x=%d on content row; expected widgets to override hints", x)
			break
		}
	}
}
