package graphics

import (
	"image"
	"io"
	"sync"

	"github.com/framegrace/texelui/core"
)

const maxChunkSize = 4096

type KittyProvider struct {
	mu      sync.Mutex
	nextID  uint32
	pending []kittyCommand
}

type kittyCommand struct{}

func NewKittyProvider() *KittyProvider {
	return &KittyProvider{nextID: 1}
}

func (p *KittyProvider) Capability() core.GraphicsCapability {
	return core.GraphicsKitty
}

func (p *KittyProvider) CreateSurface(width, height int) core.ImageSurface {
	// Stub -- will be implemented in Task 2
	return &kittySurfaceStub{id: 1, buf: image.NewRGBA(image.Rect(0, 0, width, height))}
}

func (p *KittyProvider) Reset() {
	// Stub -- will be implemented in Task 2
}

func (p *KittyProvider) Flush(w io.Writer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = nil
	return nil
}

type kittySurfaceStub struct {
	id  uint32
	buf *image.RGBA
}

func (s *kittySurfaceStub) ID() uint32          { return s.id }
func (s *kittySurfaceStub) Buffer() *image.RGBA { return s.buf }
func (s *kittySurfaceStub) Update() error       { return nil }
func (s *kittySurfaceStub) Place(p *core.Painter, rect core.Rect, zIndex int) {}
func (s *kittySurfaceStub) Delete()             {}

func chunkString(s string, n int) []string {
	if len(s) <= n {
		return []string{s}
	}
	var chunks []string
	for len(s) > n {
		chunks = append(chunks, s[:n])
		s = s[n:]
	}
	if len(s) > 0 {
		chunks = append(chunks, s)
	}
	return chunks
}
