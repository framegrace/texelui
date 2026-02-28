package graphics

import (
	"image"
	"sync"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

type rgb struct{ R, G, B uint8 }

// halfBlockSurface implements core.ImageSurface for the half-block provider.
type halfBlockSurface struct {
	id  uint32
	buf *image.RGBA
}

func (s *halfBlockSurface) ID() uint32          { return s.id }
func (s *halfBlockSurface) Buffer() *image.RGBA { return s.buf }
func (s *halfBlockSurface) Update() error       { return nil }

func (s *halfBlockSurface) Place(p *core.Painter, rect core.Rect, zIndex int) {
	if s.buf == nil {
		return
	}
	imgBounds := s.buf.Bounds()
	imgW := imgBounds.Dx()
	imgH := imgBounds.Dy()

	pixW := rect.W
	pixH := rect.H * 2

	for cy := range rect.H {
		for cx := range rect.W {
			topPixY := cy * 2
			botPixY := cy*2 + 1

			topColor := sampleRGBA(s.buf, cx, topPixY, pixW, pixH, imgW, imgH)
			botColor := sampleRGBA(s.buf, cx, botPixY, pixW, pixH, imgW, imgH)

			style := tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(int32(topColor.R), int32(topColor.G), int32(topColor.B))).
				Background(tcell.NewRGBColor(int32(botColor.R), int32(botColor.G), int32(botColor.B)))

			p.SetCell(rect.X+cx, rect.Y+cy, '\u2580', style)
		}
	}
}

func (s *halfBlockSurface) Delete() {
	s.buf = nil
}

// sampleRGBA maps output pixel coordinates to source image using nearest-neighbor.
func sampleRGBA(img *image.RGBA, cx, py, pixW, pixH, imgW, imgH int) rgb {
	imgX := cx * imgW / pixW
	imgY := py * imgH / pixH
	if imgX >= imgW {
		imgX = imgW - 1
	}
	if imgY >= imgH {
		imgY = imgH - 1
	}
	r, g, b, _ := img.At(imgX, imgY).RGBA()
	return rgb{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
}

// HalfBlockProvider renders images using Unicode half-block characters.
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
	return &halfBlockSurface{
		id:  id,
		buf: image.NewRGBA(image.Rect(0, 0, width, height)),
	}
}

// Reset is a no-op for half-block (cell-based rendering is cleared by screen.Clear).
func (p *HalfBlockProvider) Reset() {}
