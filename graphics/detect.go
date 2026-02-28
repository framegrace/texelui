package graphics

import (
	"os"
	"strings"

	"github.com/framegrace/texelui/core"
)

// knownKittyTerminals lists TERM_PROGRAM values for terminals that support
// the Kitty graphics protocol.
var knownKittyTerminals = []string{
	"kitty",
	"WezTerm",
	"ghostty",
}

// DetectCapability checks whether the terminal supports the Kitty graphics
// protocol by inspecting environment variables. Returns GraphicsKitty if
// supported, GraphicsHalfBlock otherwise.
//
// Checked variables: TERM_PROGRAM (kitty, WezTerm, ghostty),
// TERM (xterm-kitty), KITTY_WINDOW_ID.
func DetectCapability() core.GraphicsCapability {
	termProgram := os.Getenv("TERM_PROGRAM")
	for _, name := range knownKittyTerminals {
		if strings.EqualFold(termProgram, name) {
			return core.GraphicsKitty
		}
	}

	term := os.Getenv("TERM")
	if strings.Contains(term, "kitty") {
		return core.GraphicsKitty
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return core.GraphicsKitty
	}

	return core.GraphicsHalfBlock
}
