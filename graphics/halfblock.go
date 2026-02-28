package graphics

import (
	"image"
	"sync"

	"github.com/framegrace/texelui/core"
)

type HalfBlockProvider struct {
	mu     sync.Mutex
	nextID uint32
}

func NewHalfBlockProvider() *HalfBlockProvider {
	return &HalfBlockProvider{nextID: 1}
}

func (p *HalfBlockProvider) Capability() core.GraphicsCapability {
	return core.GraphicsHalfBlock
}

func (p *HalfBlockProvider) CreateSurface(width, height int) core.ImageSurface {
	p.mu.Lock()
	defer p.mu.Unlock()
	id := p.nextID
	p.nextID++
	return &halfBlockSurfaceStub{id: id, buf: image.NewRGBA(image.Rect(0, 0, width, height))}
}

func (p *HalfBlockProvider) Reset() {}

type halfBlockSurfaceStub struct {
	id  uint32
	buf *image.RGBA
}

func (s *halfBlockSurfaceStub) ID() uint32          { return s.id }
func (s *halfBlockSurfaceStub) Buffer() *image.RGBA { return s.buf }
func (s *halfBlockSurfaceStub) Update() error       { return nil }
func (s *halfBlockSurfaceStub) Place(p *core.Painter, rect core.Rect, zIndex int) {}
func (s *halfBlockSurfaceStub) Delete()             { s.buf = nil }
