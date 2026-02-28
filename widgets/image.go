package widgets

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

type Image struct {
	core.BaseWidget
	decoded  image.Image
	valid    bool
	style    tcell.Style
	altText  string
	surface  core.ImageSurface
	surfaceW int
	surfaceH int
	uploaded bool
	inv      func(core.Rect)
}

func NewImage(imgData []byte, altText string) *Image {
	tm := theme.Get()
	fg := tm.GetSemanticColor("text.primary")
	bg := tm.GetSemanticColor("bg.surface")

	img := &Image{
		altText: altText,
		style:   tcell.StyleDefault.Foreground(fg).Background(bg),
	}
	img.SetFocusable(false)

	decoded, _, err := image.Decode(bytes.NewReader(imgData))
	if err == nil {
		img.decoded = decoded
		img.valid = true
	}
	return img
}

func (img *Image) Draw(p *core.Painter) {
	if img.Rect.W == 0 || img.Rect.H == 0 {
		return
	}
	if !img.valid {
		img.drawAltText(p)
		return
	}

	gp := p.GraphicsProvider()
	if gp == nil || gp.Capability() < core.GraphicsHalfBlock {
		img.drawAltText(p)
		return
	}

	img.ensureSurface(gp)
	img.surface.Place(p, img.Rect, -1)
}

func (img *Image) ensureSurface(gp core.GraphicsProvider) {
	needsNew := img.surface == nil ||
		img.surfaceW != img.decoded.Bounds().Dx() ||
		img.surfaceH != img.decoded.Bounds().Dy()

	if needsNew {
		if img.surface != nil {
			img.surface.Delete()
		}
		bounds := img.decoded.Bounds()
		img.surfaceW = bounds.Dx()
		img.surfaceH = bounds.Dy()
		img.surface = gp.CreateSurface(img.surfaceW, img.surfaceH)
		img.uploaded = false
	}

	if !img.uploaded {
		buf := img.surface.Buffer()
		for y := range img.surfaceH {
			for x := range img.surfaceW {
				buf.Set(x, y, img.decoded.At(img.decoded.Bounds().Min.X+x, img.decoded.Bounds().Min.Y+y))
			}
		}
		img.surface.Update()
		img.uploaded = true
	}
}

func (img *Image) drawAltText(p *core.Painter) {
	text := fmt.Sprintf("[img: %s]", img.altText)
	runes := []rune(text)
	for i, r := range runes {
		if i >= img.Rect.W {
			break
		}
		p.SetCell(img.Rect.X+i, img.Rect.Y, r, img.style)
	}
}

func (img *Image) SetInvalidator(fn func(core.Rect)) { img.inv = fn }
