package graphics

import "github.com/framegrace/texelui/core"

// HalfBlockProvider signals that only Unicode half-block art is available.
// All image operations are no-ops; the Image widget handles rendering itself.
type HalfBlockProvider struct{}

func NewHalfBlockProvider() *HalfBlockProvider                    { return &HalfBlockProvider{} }
func (p *HalfBlockProvider) Capability() core.GraphicsCapability  { return core.GraphicsHalfBlock }
func (p *HalfBlockProvider) PlaceImage(core.ImagePlacement) error { return nil }
func (p *HalfBlockProvider) DeleteImage(uint32)                   {}
func (p *HalfBlockProvider) DeleteAll()                           {}
