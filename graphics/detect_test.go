package graphics

import (
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestDetectKittyViaTermProgram(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected core.GraphicsCapability
	}{
		{"kitty lowercase", "kitty", core.GraphicsKitty},
		{"kitty mixed case", "Kitty", core.GraphicsKitty},
		{"WezTerm", "WezTerm", core.GraphicsKitty},
		{"wezterm lowercase", "wezterm", core.GraphicsKitty},
		{"ghostty", "ghostty", core.GraphicsKitty},
		{"Ghostty mixed", "Ghostty", core.GraphicsKitty},
		{"unknown terminal", "xterm", core.GraphicsHalfBlock},
		{"empty", "", core.GraphicsHalfBlock},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TERM_PROGRAM", tt.value)
			t.Setenv("TERM", "xterm-256color")
			t.Setenv("KITTY_WINDOW_ID", "")

			got := DetectCapability()
			if got != tt.expected {
				t.Errorf("TERM_PROGRAM=%q: got %d, want %d", tt.value, got, tt.expected)
			}
		})
	}
}

func TestDetectKittyViaTERM(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-kitty")

	got := DetectCapability()
	if got != core.GraphicsKitty {
		t.Errorf("TERM=xterm-kitty: got %d, want GraphicsKitty", got)
	}
}

func TestDetectKittyViaWindowID(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("KITTY_WINDOW_ID", "1")

	got := DetectCapability()
	if got != core.GraphicsKitty {
		t.Errorf("KITTY_WINDOW_ID=1: got %d, want GraphicsKitty", got)
	}
}

func TestDetectFallbackToHalfBlock(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("KITTY_WINDOW_ID", "")

	got := DetectCapability()
	if got != core.GraphicsHalfBlock {
		t.Errorf("no kitty env: got %d, want GraphicsHalfBlock", got)
	}
}
