package graphics

import (
	"bytes"
	"io"
	"time"

	"github.com/framegrace/texelui/core"
)

const detectionTimeout = 100 * time.Millisecond

// queryPayload is a 1x1 pixel, RGB (f=24), query action.
// The 4 bytes "AAAA" decode to 3 zero-value RGB bytes (one pixel).
var queryPayload = []byte("\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\")

// successResponse is the expected OK reply from the terminal.
var successResponse = []byte("\x1b_Gi=31;OK\x1b\\")

// DetectCapability probes the terminal for Kitty graphics support.
// It writes a query to w and reads the response from r with a timeout.
// Returns GraphicsKitty if supported, GraphicsHalfBlock otherwise.
func DetectCapability(rw io.ReadWriter) core.GraphicsCapability {
	return detectFromReadWriter(rw, rw)
}

// detectFromReadWriter allows separate reader/writer for testing.
func detectFromReadWriter(r io.Reader, w io.Writer) core.GraphicsCapability {
	// Send query
	if _, err := w.Write(queryPayload); err != nil {
		return core.GraphicsHalfBlock
	}

	// Read response with timeout via goroutine.
	// Note: on timeout, this goroutine may remain blocked on Read until
	// the next TTY input arrives or the program exits. This is acceptable
	// because DetectCapability is called once at startup.
	responseCh := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 256)
		var collected []byte
		for {
			n, err := r.Read(buf)
			if n > 0 {
				collected = append(collected, buf[:n]...)
				if bytes.Contains(collected, successResponse) {
					responseCh <- collected
					return
				}
			}
			if err != nil {
				responseCh <- collected
				return
			}
		}
	}()

	select {
	case resp := <-responseCh:
		if bytes.Contains(resp, successResponse) {
			return core.GraphicsKitty
		}
		return core.GraphicsHalfBlock
	case <-time.After(detectionTimeout):
		return core.GraphicsHalfBlock
	}
}
