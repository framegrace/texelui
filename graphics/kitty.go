package graphics

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"sync"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

const maxChunkSize = 4096 // max base64 bytes per APC sequence

type kittyCommandType int

const (
	cmdTransmit        kittyCommandType = iota // a=t (upload, no display)
	cmdPut                                     // a=p (display from cache)
	cmdDeleteID                                // a=d,d=I (delete data+placements by ID)
	cmdResetPlacements                         // a=d,d=a (delete all placements, keep data)
)

type kittyCommand struct {
	cmdType kittyCommandType
	id      uint32
	rect    core.Rect
	zIndex  int
	pngData []byte // only for transmit commands
}

// kittySurface implements core.ImageSurface for the Kitty provider.
type kittySurface struct {
	provider *KittyProvider
	id       uint32
	buf      *image.RGBA
}

func (s *kittySurface) ID() uint32          { return s.id }
func (s *kittySurface) Buffer() *image.RGBA { return s.buf }

func (s *kittySurface) Update() error {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, s.buf); err != nil {
		return fmt.Errorf("png encode: %w", err)
	}
	s.provider.mu.Lock()
	defer s.provider.mu.Unlock()
	s.provider.pending = append(s.provider.pending, kittyCommand{
		cmdType: cmdTransmit,
		id:      s.id,
		pngData: pngBuf.Bytes(),
	})
	return nil
}

func (s *kittySurface) Place(p *core.Painter, rect core.Rect, zIndex int) {
	p.Fill(rect, ' ', tcell.StyleDefault)
	s.provider.mu.Lock()
	defer s.provider.mu.Unlock()
	s.provider.pending = append(s.provider.pending, kittyCommand{
		cmdType: cmdPut,
		id:      s.id,
		rect:    rect,
		zIndex:  zIndex,
	})
}

func (s *kittySurface) Delete() {
	s.provider.mu.Lock()
	defer s.provider.mu.Unlock()
	s.provider.pending = append(s.provider.pending, kittyCommand{
		cmdType: cmdDeleteID,
		id:      s.id,
	})
	s.buf = nil
}

// KittyProvider renders images via the Kitty graphics protocol.
type KittyProvider struct {
	mu      sync.Mutex
	nextID  uint32
	pending []kittyCommand
}

func NewKittyProvider() *KittyProvider {
	return &KittyProvider{nextID: 1}
}

func (p *KittyProvider) Capability() core.GraphicsCapability {
	return core.GraphicsKitty
}

func (p *KittyProvider) CreateSurface(width, height int) core.ImageSurface {
	p.mu.Lock()
	defer p.mu.Unlock()
	id := p.nextID
	p.nextID++
	return &kittySurface{
		provider: p,
		id:       id,
		buf:      image.NewRGBA(image.Rect(0, 0, width, height)),
	}
}

func (p *KittyProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(p.pending, kittyCommand{cmdType: cmdResetPlacements})
}

func (p *KittyProvider) Flush(w io.Writer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, cmd := range p.pending {
		var err error
		switch cmd.cmdType {
		case cmdTransmit:
			err = writeTransmit(w, cmd.id, cmd.pngData)
		case cmdPut:
			err = writePut(w, cmd.id, cmd.rect, cmd.zIndex)
		case cmdDeleteID:
			err = writeDeleteByID(w, cmd.id)
		case cmdResetPlacements:
			err = writeResetPlacements(w)
		}
		if err != nil {
			p.pending = nil
			return err
		}
	}
	p.pending = nil
	return nil
}

func writeTransmit(w io.Writer, id uint32, pngData []byte) error {
	encoded := base64.StdEncoding.EncodeToString(pngData)
	chunks := chunkString(encoded, maxChunkSize)
	for i, chunk := range chunks {
		more := 1
		if i == len(chunks)-1 {
			more = 0
		}
		var err error
		if i == 0 {
			_, err = fmt.Fprintf(w,
				"\x1b_Ga=t,f=100,t=d,i=%d,q=2,m=%d;%s\x1b\\",
				id, more, chunk)
		} else {
			_, err = fmt.Fprintf(w, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func writePut(w io.Writer, id uint32, rect core.Rect, zIndex int) error {
	if _, err := fmt.Fprintf(w, "\x1b[%d;%dH", rect.Y+1, rect.X+1); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w,
		"\x1b_Ga=p,i=%d,c=%d,r=%d,z=%d,q=2;\x1b\\",
		id, rect.W, rect.H, zIndex)
	return err
}

func writeDeleteByID(w io.Writer, id uint32) error {
	_, err := fmt.Fprintf(w, "\x1b_Ga=d,d=I,i=%d,q=2;\x1b\\", id)
	return err
}

func writeResetPlacements(w io.Writer) error {
	_, err := fmt.Fprintf(w, "\x1b_Ga=d,d=a,q=2;\x1b\\")
	return err
}

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
