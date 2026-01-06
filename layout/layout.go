package layout

import "github.com/framegrace/texelui/core"

// Absolute performs no changes; widgets manage their own positions.
type Absolute struct{}

func (Absolute) Apply(container core.Rect, children []core.Widget) {}
