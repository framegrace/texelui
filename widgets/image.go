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

// Image renders an image using half-block characters for block art,
// or via Kitty graphics protocol when a capable provider is available.
// Each cell represents two vertical pixels using the upper-half-block character
// (\u2580), with the foreground color for the top pixel and background for the
// bottom pixel.
type Image struct {
	core.BaseWidget
	imgData  []byte
	rawImgBytes []byte // raw image bytes (PNG/JPEG/GIF) for Kitty protocol
	altText  string
	decoded  image.Image
	valid    bool
	style    tcell.Style
	imageID  uint32    // Kitty image ID (0 = not yet allocated)
	lastRect core.Rect // last placed rect for move detection

	// Invalidation callback
	inv func(core.Rect)
}

// NewImage creates an Image widget from raw image bytes (PNG, JPEG, or GIF).
// If the data cannot be decoded, the widget falls back to displaying the alt
// text placeholder.
func NewImage(imgData []byte, altText string) *Image {
	tm := theme.Get()
	fg := tm.GetSemanticColor("text.primary")
	bg := tm.GetSemanticColor("bg.surface")

	img := &Image{
		imgData: imgData,
		altText: altText,
		style:   tcell.StyleDefault.Foreground(fg).Background(bg),
	}
	img.SetFocusable(false)

	// Try to decode the image. On success, keep the raw PNG bytes
	// for Kitty graphics (f=100) but discard the original reference.
	decoded, _, err := image.Decode(bytes.NewReader(imgData))
	if err == nil {
		img.decoded = decoded
		img.valid = true
		img.rawImgBytes = make([]byte, len(imgData))
		copy(img.rawImgBytes, imgData)
		img.imgData = nil
	}

	return img
}

// Draw renders the image using the best available method: Kitty graphics
// protocol when a capable provider is present, otherwise half-block art.
// Falls back to alt text if the image data is invalid.
func (img *Image) Draw(p *core.Painter) {
	if img.Rect.W == 0 || img.Rect.H == 0 {
		return
	}
	if !img.valid {
		img.drawAltText(p)
		return
	}

	gp := p.GraphicsProvider()
	if gp != nil && gp.Capability() >= core.GraphicsKitty {
		img.drawKitty(p, gp)
		return
	}

	img.drawHalfBlock(p)
}

// drawKitty renders the image via the Kitty graphics protocol.
func (img *Image) drawKitty(p *core.Painter, gp core.GraphicsProvider) {
	// Fill region with spaces so tcell clears the area
	p.Fill(img.Rect, ' ', img.style)

	// Delete old placement if position/size changed
	if img.imageID != 0 && img.lastRect != img.Rect {
		gp.DeleteImage(img.imageID)
		img.imageID = 0
	}

	// Allocate ID on first use
	if img.imageID == 0 {
		if alloc, ok := gp.(interface{ AllocateID() uint32 }); ok {
			img.imageID = alloc.AllocateID()
		} else {
			img.imageID = 1
		}
	}

	gp.PlaceImage(core.ImagePlacement{
		ID:      img.imageID,
		Rect:    img.Rect,
		ImgData: img.rawImgBytes,
		ZIndex:  -1,
	})
	img.lastRect = img.Rect
}

// drawHalfBlock renders the image as Unicode half-block art.
func (img *Image) drawHalfBlock(p *core.Painter) {
	imgBounds := img.decoded.Bounds()
	imgW := imgBounds.Dx()
	imgH := imgBounds.Dy()

	// Each cell = 1 column width, 2 pixel rows (using half-block).
	pixW := img.Rect.W
	pixH := img.Rect.H * 2

	for cy := 0; cy < img.Rect.H; cy++ {
		for cx := 0; cx < img.Rect.W; cx++ {
			topPixY := cy * 2
			botPixY := cy*2 + 1

			topColor := img.sampleColor(cx, topPixY, pixW, pixH, imgW, imgH)
			botColor := img.sampleColor(cx, botPixY, pixW, pixH, imgW, imgH)

			style := tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(int32(topColor.r), int32(topColor.g), int32(topColor.b))).
				Background(tcell.NewRGBColor(int32(botColor.r), int32(botColor.g), int32(botColor.b)))

			p.SetCell(img.Rect.X+cx, img.Rect.Y+cy, '\u2580', style)
		}
	}
}

// drawAltText renders a placeholder when image data is invalid.
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

type rgb struct{ r, g, b uint8 }

// sampleColor maps a pixel coordinate in the output space to the source image
// using nearest-neighbor scaling.
func (img *Image) sampleColor(cx, py, pixW, pixH, imgW, imgH int) rgb {
	imgX := cx * imgW / pixW
	imgY := py * imgH / pixH

	if imgX >= imgW {
		imgX = imgW - 1
	}
	if imgY >= imgH {
		imgY = imgH - 1
	}

	bounds := img.decoded.Bounds()
	r, g, b, _ := img.decoded.At(bounds.Min.X+imgX, bounds.Min.Y+imgY).RGBA()
	return rgb{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
}

// SetInvalidator allows the UI manager to inject a dirty-region invalidator.
func (img *Image) SetInvalidator(fn func(core.Rect)) { img.inv = fn }
