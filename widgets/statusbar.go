// Copyright 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: texelui/widgets/statusbar.go
// Summary: Global status bar widget with key hints and timed messages.

package widgets

import (
	"sync"
	"time"

	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

// MessageLevel defines the priority/styling of status messages.
type MessageLevel int

const (
	MessageInfo MessageLevel = iota
	MessageSuccess
	MessageWarning
	MessageError
)

// TimedMessage represents a message with expiration and priority.
type TimedMessage struct {
	Text      string
	Level     MessageLevel
	ExpiresAt time.Time
}

// StatusBar displays key hints (left) and timed messages (right).
// It implements FocusObserver to automatically update key hints when focus changes.
// IMPORTANT: Call Stop() before discarding a StatusBar to prevent goroutine leaks.
type StatusBar struct {
	core.BaseWidget

	mu            sync.Mutex
	leftText      string         // Current key hints (formatted)
	leftWidgets   []core.Widget  // Child widgets for left side (overrides leftText)
	messages      []TimedMessage // Message queue, highest priority shown
	focusedWidget core.Widget    // Currently focused widget for hint extraction
	hoverHelp     string         // Currently displayed hover help text (empty = none)

	inv      func(core.Rect)
	ticker   *time.Ticker
	stopCh   chan struct{}
	stopped  bool        // Tracks if Stop() has been called
	notifier chan<- bool // Refresh notifier from UIManager

	// DefaultMessageDuration is the default duration for messages
	DefaultMessageDuration time.Duration

	// ShowSeparator controls whether the top separator line is drawn.
	// When false, the status bar is 1 row (content only).
	ShowSeparator bool
}

// NewStatusBar creates a new status bar widget.
// The status bar is 2 rows: 1 for separator line, 1 for content.
// Position defaults to 0,0 and width to 1.
// Use SetPosition and Resize to adjust after adding to a layout.
func NewStatusBar() *StatusBar {
	sb := &StatusBar{
		messages:               make([]TimedMessage, 0, 10),
		stopCh:                 make(chan struct{}),
		DefaultMessageDuration: 3 * time.Second,
		ShowSeparator:          true,
	}
	sb.SetPosition(0, 0)
	sb.Resize(1, 2)        // 1 separator + 1 content row
	sb.SetFocusable(false) // Status bar never receives focus
	return sb
}

// Start begins the background ticker for message expiration.
// Should be called after the status bar is added to UIManager.
func (s *StatusBar) Start() {
	s.mu.Lock()
	if s.ticker != nil {
		s.mu.Unlock()
		return // Already started
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	s.ticker = ticker
	s.mu.Unlock()

	go func() {
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				if s.expireMessages() {
					s.invalidate()
				}
			}
		}
	}()
}

// Stop stops the background ticker.
// Must be called before discarding a StatusBar to prevent goroutine leaks.
// Safe to call multiple times.
func (s *StatusBar) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return // Already stopped
	}
	s.stopped = true

	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}

	close(s.stopCh)
}

// SetRefreshNotifier sets the refresh notifier for triggering redraws.
func (s *StatusBar) SetRefreshNotifier(ch chan<- bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifier = ch
}

// OnFocusChanged implements core.FocusObserver.
// Called by UIManager when the focused widget changes.
func (s *StatusBar) OnFocusChanged(focused core.Widget) {
	s.mu.Lock()
	s.focusedWidget = focused
	s.updateKeyHintsLocked()
	s.mu.Unlock()
	s.invalidate()
}

// updateKeyHintsLocked extracts and formats key hints from the focused widget.
// Must be called with s.mu held.
func (s *StatusBar) updateKeyHintsLocked() {
	if s.focusedWidget == nil {
		s.leftText = ""
		return
	}

	// Find the most deeply focused widget (handles containers like TabLayout)
	deepWidget := core.FindDeepFocused(s.focusedWidget)
	if deepWidget == nil {
		deepWidget = s.focusedWidget
	}

	// Get widget's own key hints
	var hints []core.KeyHint
	if khp, ok := deepWidget.(core.KeyHintsProvider); ok {
		hints = khp.GetKeyHints()
	}

	// Add focus navigation hints if inside a focus cycler
	// Skip hints for keys the widget already defines (avoid duplicates like Tab:Content + Tab:Next)
	if s.hasFocusCycling() {
		if !hasKeyHint(hints, "Tab") {
			hints = append(hints, core.KeyHint{Key: "Tab", Label: "Next"})
		}
		if !hasKeyHint(hints, "S-Tab") {
			hints = append(hints, core.KeyHint{Key: "S-Tab", Label: "Prev"})
		}
	}

	s.leftText = core.FormatKeyHints(hints)
}

// hasKeyHint checks if hints already contains a hint for the given key.
func hasKeyHint(hints []core.KeyHint, key string) bool {
	for _, h := range hints {
		if h.Key == key {
			return true
		}
	}
	return false
}

// hasFocusCycling checks if focus cycling is available from the focused widget.
func (s *StatusBar) hasFocusCycling() bool {
	if s.focusedWidget == nil {
		return false
	}

	// Check if the root focused widget or any ancestor supports focus cycling
	if _, ok := s.focusedWidget.(core.FocusCycler); ok {
		return true
	}

	// Also check if it's a child container with focus cycling
	if cc, ok := s.focusedWidget.(core.ChildContainer); ok {
		_ = cc // Focus cycling is typically available in containers
		return true
	}

	return false
}

// ShowMessage displays an info message with the default duration.
func (s *StatusBar) ShowMessage(text string) {
	s.ShowMessageWithDuration(text, s.DefaultMessageDuration)
}

// ShowMessageWithDuration displays an info message with a custom duration.
func (s *StatusBar) ShowMessageWithDuration(text string, duration time.Duration) {
	s.addMessage(text, MessageInfo, duration)
}

// ShowSuccess displays a success message (green) with the default duration.
func (s *StatusBar) ShowSuccess(text string) {
	s.ShowSuccessWithDuration(text, s.DefaultMessageDuration)
}

// ShowSuccessWithDuration displays a success message with a custom duration.
func (s *StatusBar) ShowSuccessWithDuration(text string, duration time.Duration) {
	s.addMessage(text, MessageSuccess, duration)
}

// ShowWarning displays a warning message (yellow) with the default duration.
func (s *StatusBar) ShowWarning(text string) {
	s.ShowWarningWithDuration(text, s.DefaultMessageDuration)
}

// ShowWarningWithDuration displays a warning message with a custom duration.
func (s *StatusBar) ShowWarningWithDuration(text string, duration time.Duration) {
	s.addMessage(text, MessageWarning, duration)
}

// ShowError displays an error message (red) with the default duration.
func (s *StatusBar) ShowError(text string) {
	s.ShowErrorWithDuration(text, s.DefaultMessageDuration)
}

// ShowErrorWithDuration displays an error message with a custom duration.
func (s *StatusBar) ShowErrorWithDuration(text string, duration time.Duration) {
	s.addMessage(text, MessageError, duration)
}

// addMessage adds a message to the queue.
func (s *StatusBar) addMessage(text string, level MessageLevel, duration time.Duration) {
	s.mu.Lock()

	msg := TimedMessage{
		Text:      text,
		Level:     level,
		ExpiresAt: time.Now().Add(duration),
	}

	// Limit queue size to prevent memory growth
	if len(s.messages) >= 10 {
		// Remove oldest message
		s.messages = s.messages[1:]
	}

	s.messages = append(s.messages, msg)
	s.mu.Unlock()

	// Call invalidate after releasing the lock to avoid deadlock
	s.invalidate()
}

// expireMessages removes expired messages and returns true if any were removed.
func (s *StatusBar) expireMessages() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	originalLen := len(s.messages)

	// Filter out expired messages
	filtered := s.messages[:0]
	for _, msg := range s.messages {
		if msg.ExpiresAt.After(now) {
			filtered = append(filtered, msg)
		}
	}
	s.messages = filtered

	return len(s.messages) != originalLen
}

// getActiveMessage returns the highest priority non-expired message.
// Returns nil if no messages are active.
func (s *StatusBar) getActiveMessage() *TimedMessage {
	now := time.Now()
	var best *TimedMessage

	for i := range s.messages {
		msg := &s.messages[i]
		if msg.ExpiresAt.After(now) {
			if best == nil || msg.Level > best.Level {
				best = msg
			}
		}
	}

	return best
}

// Draw renders the status bar.
func (s *StatusBar) Draw(p *core.Painter) {
	tm := theme.Get()
	fg := tm.GetSemanticColor("text.primary")
	bg := tm.GetSemanticColor("bg.surface")
	sepFg := tm.GetSemanticColor("border.default")

	bgStyle := tcell.StyleDefault.Foreground(fg).Background(bg)
	sepStyle := tcell.StyleDefault.Foreground(sepFg).Background(bg)

	contentY := s.Rect.Y
	if s.ShowSeparator {
		// Row 0: Separator line
		for x := 0; x < s.Rect.W; x++ {
			p.SetCell(s.Rect.X+x, s.Rect.Y, '─', sepStyle)
		}
		contentY++
	}
	for x := 0; x < s.Rect.W; x++ {
		p.SetCell(s.Rect.X+x, contentY, ' ', bgStyle)
	}

	s.mu.Lock()
	hasLeftWidgets := len(s.leftWidgets) > 0
	activeMsg := s.getActiveMessage()

	var leftUsedWidth int
	if hasLeftWidgets {
		// Layout and copy child widgets under the lock
		leftUsedWidth = s.layoutLeftWidgets()
		widgets := make([]core.Widget, len(s.leftWidgets))
		copy(widgets, s.leftWidgets)
		s.mu.Unlock()

		// Draw each widget outside the lock
		for _, w := range widgets {
			w.Draw(p)
		}
	} else {
		// Refresh key hints on every draw to catch internal focus changes
		// (e.g., TabLayout switching between tab bar and content)
		s.updateKeyHintsLocked()
		leftText := s.leftText
		s.mu.Unlock()

		// Get rune slices for proper UTF-8 handling
		leftRunes := []rune(leftText)
		var rightText string
		var rightRunes []rune
		if activeMsg != nil {
			rightText = activeMsg.Text
			rightRunes = []rune(rightText)
		}

		// Calculate available space
		availableWidth := s.Rect.W - 2 // 1 char padding on each side

		// Only truncate key hints if there's a message that needs space
		if len(rightRunes) > 0 {
			// Reserve space for message + gap (3 chars gap between hints and message)
			msgSpace := len(rightRunes) + 3
			maxLeft := availableWidth - msgSpace
			if maxLeft < 1 {
				maxLeft = 1
			}
			if len(leftRunes) > maxLeft {
				if maxLeft > 1 {
					leftText = string(leftRunes[:maxLeft-1]) + "…"
					leftRunes = []rune(leftText)
				} else {
					leftText = "…"
					leftRunes = []rune(leftText)
				}
			}
		} else {
			// No message - only truncate if hints exceed available width
			if len(leftRunes) > availableWidth {
				if availableWidth > 1 {
					leftText = string(leftRunes[:availableWidth-1]) + "…"
					leftRunes = []rune(leftText)
				} else {
					leftText = "…"
					leftRunes = []rune(leftText)
				}
			}
		}

		leftUsedWidth = len(leftRunes)

		// Draw left text (key hints) - dimmed style
		if leftText != "" {
			hintFg := tm.GetSemanticColor("text.secondary")
			if hintFg == tcell.ColorDefault {
				hintFg = tcell.ColorGray
			}
			hintStyle := tcell.StyleDefault.Foreground(hintFg).Background(bg)
			p.DrawText(s.Rect.X+1, contentY, leftText, hintStyle)
		}
	}

	// Draw right text (messages) - right-aligned with level-based coloring
	if activeMsg != nil {
		rightText := activeMsg.Text
		rightRunes := []rune(rightText)

		if len(rightRunes) > 0 {
			msgStyle := s.getMessageStyle(activeMsg.Level, bg)

			// Calculate right-aligned position
			rightX := s.Rect.X + s.Rect.W - len(rightRunes) - 1

			// Check if message needs truncation
			minX := s.Rect.X + leftUsedWidth + 3
			if rightX < minX {
				maxLen := s.Rect.W - leftUsedWidth - 4
				if maxLen > 3 && maxLen-1 < len(rightRunes) {
					rightText = string(rightRunes[:maxLen-1]) + "…"
					rightRunes = []rune(rightText)
					rightX = s.Rect.X + s.Rect.W - len(rightRunes) - 1
				} else if maxLen <= 3 {
					rightText = "" // Not enough space
				}
			}

			if rightText != "" {
				p.DrawText(rightX, contentY, rightText, msgStyle)
			}
		}
	}
}

// getMessageStyle returns the style for a message based on its level.
func (s *StatusBar) getMessageStyle(level MessageLevel, bg tcell.Color) tcell.Style {
	tm := theme.Get()

	var fg tcell.Color
	switch level {
	case MessageSuccess:
		fg = tm.GetSemanticColor("status.success")
		if fg == tcell.ColorDefault {
			fg = tcell.ColorGreen
		}
	case MessageWarning:
		fg = tm.GetSemanticColor("status.warning")
		if fg == tcell.ColorDefault {
			fg = tcell.ColorYellow
		}
	case MessageError:
		fg = tm.GetSemanticColor("status.error")
		if fg == tcell.ColorDefault {
			fg = tcell.ColorRed
		}
	default: // MessageInfo
		fg = tm.GetSemanticColor("text.primary")
	}

	return tcell.StyleDefault.Foreground(fg).Background(bg)
}

// SetInvalidator implements core.InvalidationAware.
func (s *StatusBar) SetInvalidator(fn func(core.Rect)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inv = fn
}

// invalidate marks the status bar as needing redraw.
func (s *StatusBar) invalidate() {
	s.mu.Lock()
	inv := s.inv
	notifier := s.notifier
	s.mu.Unlock()

	if inv != nil {
		inv(s.Rect)
	}

	// Also trigger refresh notifier if available
	if notifier != nil {
		select {
		case notifier <- true:
		default:
		}
	}
}

// ClearMessages removes all messages from the queue.
func (s *StatusBar) ClearMessages() {
	s.mu.Lock()
	s.messages = s.messages[:0]
	s.mu.Unlock()
	s.invalidate()
}

// ClearKeyHints clears the key hints display.
func (s *StatusBar) ClearKeyHints() {
	s.mu.Lock()
	s.leftText = ""
	s.mu.Unlock()
	s.invalidate()
}

// SetLeftWidgets sets widgets to display on the left side of the status bar.
// Widgets are positioned left-to-right with 1-char gaps during Draw.
// Takes priority over KeyHintsProvider hints when set.
// Pass nil to clear and revert to KeyHintsProvider behavior.
func (s *StatusBar) SetLeftWidgets(widgets []core.Widget) {
	s.mu.Lock()
	s.leftWidgets = widgets
	s.mu.Unlock()
	s.invalidate()
}

// layoutLeftWidgets positions left-side widgets sequentially on the content row.
// Returns the total width consumed (for spacing right-side messages).
// Must be called with s.mu held.
func (s *StatusBar) layoutLeftWidgets() int {
	contentY := s.Rect.Y
	if s.ShowSeparator {
		contentY++
	}
	xx := s.Rect.X + 1 // 1-char left padding
	for i, w := range s.leftWidgets {
		w.SetPosition(xx, contentY)
		ww, _ := w.Size()
		xx += ww
		if i < len(s.leftWidgets)-1 {
			xx++ // 1-char gap between widgets
		}
	}
	return xx - (s.Rect.X + 1) // total width consumed
}

// HandleMouse forwards mouse events to left-side widgets and shows
// hover help text from any widget implementing core.HelpTextProvider.
// Returns true if a widget handled the event.
func (s *StatusBar) HandleMouse(ev *tcell.EventMouse) bool {
	x, y := ev.Position()
	if !s.Rect.Contains(x, y) {
		s.clearHoverHelp()
		return false
	}

	s.mu.Lock()
	widgets := make([]core.Widget, len(s.leftWidgets))
	copy(widgets, s.leftWidgets)
	s.mu.Unlock()

	// Check hover help for all widgets (works on motion and click events)
	helpFound := false
	for _, w := range widgets {
		if w.HitTest(x, y) {
			if hp, ok := w.(core.HelpTextProvider); ok {
				if ht := hp.HelpText(); ht != "" {
					s.setHoverHelp(ht)
					helpFound = true
				}
			}
			break
		}
	}
	if !helpFound {
		s.clearHoverHelp()
	}

	// Forward click events to widgets
	for _, w := range widgets {
		if ma, ok := w.(core.MouseAware); ok {
			if ma.HandleMouse(ev) {
				s.invalidate()
				return true
			}
		}
	}
	return helpFound
}

// setHoverHelp shows help text if it differs from the current hover help.
func (s *StatusBar) setHoverHelp(text string) {
	s.mu.Lock()
	if s.hoverHelp == text {
		s.mu.Unlock()
		return
	}
	s.hoverHelp = text
	s.messages = append(s.messages, TimedMessage{
		Text:      text,
		Level:     MessageInfo,
		ExpiresAt: time.Now().Add(30 * time.Second),
	})
	s.mu.Unlock()
	s.invalidate()
}

// clearHoverHelp removes any active hover help message.
func (s *StatusBar) clearHoverHelp() {
	s.mu.Lock()
	if s.hoverHelp == "" {
		s.mu.Unlock()
		return
	}
	for i, m := range s.messages {
		if m.Text == s.hoverHelp {
			s.messages = append(s.messages[:i], s.messages[i+1:]...)
			break
		}
	}
	s.hoverHelp = ""
	s.mu.Unlock()
	s.invalidate()
}
