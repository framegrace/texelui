package core

// GraphicsCapability describes what level of image rendering a provider supports.
type GraphicsCapability int

const (
	GraphicsNone      GraphicsCapability = iota
	GraphicsHalfBlock                    // Unicode half-block art (always available)
	GraphicsKitty                        // Kitty graphics protocol (APC sequences)
)

// ImagePlacement represents an image to display at a screen position.
type ImagePlacement struct {
	ID      uint32 // unique image ID for updates/deletion
	Rect    Rect   // screen cell region to occupy
	ImgData []byte // raw PNG/JPEG/GIF bytes
	ZIndex  int    // negative = behind text
}

// GraphicsProvider abstracts image rendering capabilities.
// The runtime injects an implementation at startup.
// Widgets query Capability() to choose their rendering strategy.
type GraphicsProvider interface {
	Capability() GraphicsCapability
	PlaceImage(p ImagePlacement) error
	DeleteImage(id uint32)
	DeleteAll()
}
