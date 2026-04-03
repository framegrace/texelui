package textrender

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Config configures the text renderer.
type Config struct {
	FontPath   string // path to TTF/OTF file (required)
	CellWidth  int    // cell width in pixels (0 = derive from font metrics)
	CellHeight int    // cell height in pixels (0 = derive from font metrics)
}

// Renderer renders a cell grid to an RGBA image using a monospace font.
type Renderer struct {
	face  font.Face
	cellW int
	cellH int
	asc   int // font ascent in pixels (baseline offset from top of cell)
}

const (
	defaultFontSize = 14.0
	defaultDPI      = 72.0
)

// New creates a Renderer from the given Config. The font file at cfg.FontPath
// is loaded, parsed, and sized so that the resulting cell dimensions match any
// explicit CellWidth/CellHeight values in cfg.
func New(cfg Config) (*Renderer, error) {
	data, err := os.ReadFile(cfg.FontPath)
	if err != nil {
		return nil, fmt.Errorf("textrender: reading font file: %w", err)
	}

	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("textrender: parsing font: %w", err)
	}

	// Initial face at default size to measure metrics.
	fontSize := defaultFontSize
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     defaultDPI,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("textrender: creating font face: %w", err)
	}

	metrics := face.Metrics()
	fontCellH := (metrics.Ascent + metrics.Descent).Ceil()
	fontCellW := measureAdvance(face, 'M')

	// If an explicit cell height was requested and differs from the natural
	// metrics, recompute the font size and recreate the face.
	if cfg.CellHeight > 0 && cfg.CellHeight != fontCellH {
		adjusted := defaultFontSize * float64(cfg.CellHeight) / float64(fontCellH)
		face.Close()
		face, err = opentype.NewFace(parsed, &opentype.FaceOptions{
			Size:    adjusted,
			DPI:     defaultDPI,
			Hinting: font.HintingFull,
		})
		if err != nil {
			return nil, fmt.Errorf("textrender: creating adjusted font face: %w", err)
		}
		metrics = face.Metrics()
		fontCellH = (metrics.Ascent + metrics.Descent).Ceil()
		fontCellW = measureAdvance(face, 'M')
	} else if cfg.CellHeight == 0 {
		// No explicit size; use font-derived metrics normally.
	} else {
		// Exact match on first try — no recreation needed.
	}

	cellW := fontCellW
	if cfg.CellWidth > 0 {
		cellW = cfg.CellWidth
	}
	cellH := fontCellH
	if cfg.CellHeight > 0 {
		cellH = cfg.CellHeight
	}

	asc := metrics.Ascent.Ceil()

	return &Renderer{
		face:  face,
		cellW: cellW,
		cellH: cellH,
		asc:   asc,
	}, nil
}

// measureAdvance returns the advance width of r in the face, in pixels.
func measureAdvance(f font.Face, r rune) int {
	adv, ok := f.GlyphAdvance(r)
	if !ok {
		return 8 // sensible fallback
	}
	return int(math.Ceil(float64(adv) / 64.0))
}

// Render draws the cell grid to a new RGBA image. Each row in grid must have
// the same length; if the grid is empty, a zero-size image is returned.
func (rn *Renderer) Render(grid [][]core.Cell) *image.RGBA {
	rows := len(grid)
	if rows == 0 {
		return image.NewRGBA(image.Rectangle{})
	}
	cols := len(grid[0])
	if cols == 0 {
		return image.NewRGBA(image.Rectangle{})
	}

	img := image.NewRGBA(image.Rect(0, 0, cols*rn.cellW, rows*rn.cellH))

	for gy, row := range grid {
		for gx, cell := range row {
			rn.drawCell(img, gx, gy, cell)
		}
	}

	return img
}

// drawCell renders a single Cell at grid position (gx, gy).
func (rn *Renderer) drawCell(img *image.RGBA, gx, gy int, cell core.Cell) {
	fg, bg, attrs := cell.Style.Decompose()

	// Reverse video swaps fg and bg.
	if attrs&tcell.AttrReverse != 0 {
		fg, bg = bg, fg
	}

	fgRGBA := tcellToRGBA(fg, color.RGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}) // light gray
	bgRGBA := tcellToRGBA(bg, color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}) // black

	// Dim: reduce foreground luminance.
	if attrs&tcell.AttrDim != 0 {
		fgRGBA = color.RGBA{
			R: uint8(float64(fgRGBA.R) * 0.6),
			G: uint8(float64(fgRGBA.G) * 0.6),
			B: uint8(float64(fgRGBA.B) * 0.6),
			A: fgRGBA.A,
		}
	}

	// Fill cell background.
	x0 := gx * rn.cellW
	y0 := gy * rn.cellH
	cellRect := image.Rect(x0, y0, x0+rn.cellW, y0+rn.cellH)
	draw.Draw(img, cellRect, &image.Uniform{bgRGBA}, image.Point{}, draw.Src)

	// Draw glyph (skip space, null, and other invisible runes).
	ch := cell.Ch
	if ch != 0 && ch != ' ' {
		d := font.Drawer{
			Dst:  img,
			Src:  &image.Uniform{fgRGBA},
			Face: rn.face,
			Dot:  fixed.P(x0, y0+rn.asc),
		}
		d.DrawString(string(ch))
	}

	// Underline: 1px line at the very bottom of the cell.
	if attrs&tcell.AttrUnderline != 0 {
		y := y0 + rn.cellH - 1
		for x := x0; x < x0+rn.cellW; x++ {
			img.SetRGBA(x, y, fgRGBA)
		}
	}
}

// tcellToRGBA converts a tcell.Color to color.RGBA. If the color is
// ColorDefault or ColorNone, fallback is returned.
func tcellToRGBA(c tcell.Color, fallback color.RGBA) color.RGBA {
	if c == tcell.ColorDefault || c == tcell.ColorNone {
		return fallback
	}
	r, g, b := c.RGB()
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xff}
}
