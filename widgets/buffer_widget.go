package widgets

import (
	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

// BufferWidget adapts a [][]Cell buffer as a Widget, allowing it to
// participate in a widget tree. This is useful for compositing pre-rendered
// content (e.g., terminal output) inside container widgets like Border.
type BufferWidget struct {
	core.BaseWidget
	buffer [][]core.Cell
}

// NewBufferWidget creates a widget that renders the given cell buffer.
// The widget's size is set to match the buffer dimensions.
func NewBufferWidget(buffer [][]core.Cell) *BufferWidget {
	w := &BufferWidget{buffer: buffer}
	if len(buffer) > 0 && len(buffer[0]) > 0 {
		w.Resize(len(buffer[0]), len(buffer))
	}
	return w
}

// SetBuffer updates the cell buffer to render.
// The widget is resized to match the new buffer dimensions.
func (w *BufferWidget) SetBuffer(buffer [][]core.Cell) {
	w.buffer = buffer
	if len(buffer) > 0 && len(buffer[0]) > 0 {
		w.Resize(len(buffer[0]), len(buffer))
	}
}

// Draw copies the buffer cells to the painter at the widget's position.
func (w *BufferWidget) Draw(p *core.Painter) {
	if w.buffer == nil {
		return
	}
	wx, wy := w.Position()
	ww, wh := w.Size()
	for y := 0; y < wh && y < len(w.buffer); y++ {
		row := w.buffer[y]
		for x := 0; x < ww && x < len(row); x++ {
			cell := row[x]
			ch := cell.Ch
			if ch == 0 {
				ch = ' '
			}
			style := cell.Style
			if style == (tcell.Style{}) {
				continue // skip uninitialized cells
			}
			p.SetCell(wx+x, wy+y, ch, style)
		}
	}
}
