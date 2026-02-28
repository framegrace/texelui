package graphics

import (
	"encoding/base64"
	"fmt"
	"io"
	"sync"

	"github.com/framegrace/texelui/core"
)

const maxChunkSize = 4096 // max base64 bytes per APC sequence

// kittyCommand represents a queued Kitty protocol command.
type kittyCommand struct {
	placement *core.ImagePlacement // nil for delete commands
	deleteID  uint32               // non-zero for single delete
	deleteAll bool                 // true for delete-all
}

// KittyProvider renders images via the Kitty graphics protocol.
// PlaceImage and DeleteImage queue commands; Flush writes them to the terminal.
type KittyProvider struct {
	mu      sync.Mutex
	nextID  uint32
	pending []kittyCommand
}

// NewKittyProvider creates a KittyProvider with IDs starting at 1.
func NewKittyProvider() *KittyProvider {
	return &KittyProvider{nextID: 1}
}

// Capability returns GraphicsKitty.
func (p *KittyProvider) Capability() core.GraphicsCapability {
	return core.GraphicsKitty
}

// PlaceImage queues an image placement for the next Flush.
func (p *KittyProvider) PlaceImage(pl core.ImagePlacement) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pl.ID == 0 {
		pl.ID = p.nextID
		p.nextID++
	}
	p.pending = append(p.pending, kittyCommand{placement: &pl})
	return nil
}

// DeleteImage queues deletion of a single image by ID.
func (p *KittyProvider) DeleteImage(id uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(p.pending, kittyCommand{deleteID: id})
}

// DeleteAll queues deletion of all images.
func (p *KittyProvider) DeleteAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(p.pending, kittyCommand{deleteAll: true})
}

// AllocateID returns a unique image ID for a widget to use.
func (p *KittyProvider) AllocateID() uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	id := p.nextID
	p.nextID++
	return id
}

// Flush writes all pending commands to w, then clears the queue.
func (p *KittyProvider) Flush(w io.Writer) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.flushTo(w)
}

func (p *KittyProvider) flushTo(w io.Writer) error {
	for _, cmd := range p.pending {
		var err error
		switch {
		case cmd.deleteAll:
			err = writeDeleteAll(w)
		case cmd.deleteID != 0:
			err = writeDeleteImage(w, cmd.deleteID)
		case cmd.placement != nil:
			err = writePlacement(w, cmd.placement)
		}
		if err != nil {
			p.pending = nil
			return err
		}
	}
	p.pending = nil
	return nil
}

// writePlacement writes cursor positioning + chunked APC image data.
func writePlacement(w io.Writer, pl *core.ImagePlacement) error {
	// Move cursor to placement position (1-indexed)
	_, err := fmt.Fprintf(w, "\x1b[%d;%dH", pl.Rect.Y+1, pl.Rect.X+1)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(pl.ImgData)
	chunks := chunkString(encoded, maxChunkSize)

	for i, chunk := range chunks {
		isLast := i == len(chunks)-1
		more := 1
		if isLast {
			more = 0
		}

		if i == 0 {
			// First chunk: include all control data
			_, err = fmt.Fprintf(w,
				"\x1b_Ga=T,f=100,t=d,i=%d,c=%d,r=%d,z=%d,q=2,m=%d;%s\x1b\\",
				pl.ID, pl.Rect.W, pl.Rect.H, pl.ZIndex, more, chunk,
			)
		} else {
			// Continuation chunks: only m key needed
			_, err = fmt.Fprintf(w, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func writeDeleteImage(w io.Writer, id uint32) error {
	_, err := fmt.Fprintf(w, "\x1b_Ga=d,d=I,i=%d,q=2;\x1b\\", id)
	return err
}

func writeDeleteAll(w io.Writer) error {
	_, err := fmt.Fprintf(w, "\x1b_Ga=d,d=A,q=2;\x1b\\")
	return err
}

// chunkString splits s into chunks of at most n bytes.
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
