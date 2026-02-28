package core

import "image"

// GraphicsCapability describes what level of image rendering a provider supports.
type GraphicsCapability int

const (
	GraphicsNone      GraphicsCapability = iota
	GraphicsHalfBlock                    // Unicode half-block art (always available)
	GraphicsKitty                        // Kitty graphics protocol (APC sequences)
)

// GraphicsProvider abstracts image rendering capabilities.
// The runtime injects an implementation at startup.
// Widgets query Capability() to choose their rendering strategy.
type GraphicsProvider interface {
	Capability() GraphicsCapability
	CreateSurface(width, height int) ImageSurface
	Reset() // clear all active placements (cached data preserved)
}

// ImageSurface represents an allocated image buffer that can be rendered
// into and displayed at a screen position. The app renders into Buffer(),
// calls Update() when content changes, and calls Place() every frame.
type ImageSurface interface {
	ID() uint32
	Buffer() *image.RGBA
	Update() error
	Place(p *Painter, rect Rect, zIndex int)
	Delete()
}
