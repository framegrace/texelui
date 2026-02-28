package graphics

import (
	"bytes"
	"io"
	"testing"

	"github.com/framegrace/texelui/core"
)

// mockTTY simulates a terminal for detection tests.
type mockTTY struct {
	response []byte
	written  bytes.Buffer
	readPos  int
}

func (m *mockTTY) Read(p []byte) (int, error) {
	if m.readPos >= len(m.response) {
		return 0, io.EOF
	}
	n := copy(p, m.response[m.readPos:])
	m.readPos += n
	return n, nil
}

func (m *mockTTY) Write(p []byte) (int, error) {
	return m.written.Write(p)
}

func TestDetectKittySupported(t *testing.T) {
	tty := &mockTTY{
		response: []byte("\x1b_Gi=31;OK\x1b\\"),
	}
	cap := detectFromReadWriter(tty, tty)
	if cap != core.GraphicsKitty {
		t.Errorf("expected GraphicsKitty, got %d", cap)
	}
}

func TestDetectKittyUnsupported(t *testing.T) {
	tty := &mockTTY{
		response: []byte("\x1b[?62;c"),
	}
	cap := detectFromReadWriter(tty, tty)
	if cap != core.GraphicsHalfBlock {
		t.Errorf("expected GraphicsHalfBlock fallback, got %d", cap)
	}
}

func TestDetectKittyEmpty(t *testing.T) {
	tty := &mockTTY{
		response: []byte{},
	}
	cap := detectFromReadWriter(tty, tty)
	if cap != core.GraphicsHalfBlock {
		t.Errorf("expected GraphicsHalfBlock fallback, got %d", cap)
	}
}
