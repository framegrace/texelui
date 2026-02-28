package graphics

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestKittyCreateSurface(t *testing.T) {
	p := NewKittyProvider()
	s := p.CreateSurface(100, 50)
	if s == nil {
		t.Fatal("CreateSurface returned nil")
	}
	if s.ID() == 0 {
		t.Error("expected non-zero ID")
	}
	buf := s.Buffer()
	if buf == nil {
		t.Fatal("Buffer returned nil")
	}
	if buf.Bounds().Dx() != 100 || buf.Bounds().Dy() != 50 {
		t.Errorf("buffer size: got %dx%d, want 100x50", buf.Bounds().Dx(), buf.Bounds().Dy())
	}
}

func TestKittySurfaceIDsUnique(t *testing.T) {
	p := NewKittyProvider()
	s1 := p.CreateSurface(10, 10)
	s2 := p.CreateSurface(10, 10)
	if s1.ID() == s2.ID() {
		t.Errorf("surfaces have same ID: %d", s1.ID())
	}
}

func TestKittyUpdateQueuesTransmit(t *testing.T) {
	p := NewKittyProvider()
	s := p.CreateSurface(4, 2)
	s.Buffer().Set(0, 0, color.RGBA{255, 0, 0, 255})

	if err := s.Update(); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	var buf bytes.Buffer
	if err := p.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "a=t") {
		t.Errorf("expected a=t (transmit only), got: %q", output)
	}
	if !strings.Contains(output, "f=100") {
		t.Errorf("expected f=100 (PNG), got: %q", output)
	}
	if !strings.Contains(output, "i=1") {
		t.Errorf("expected i=1, got: %q", output)
	}
}

func TestKittyPlaceQueuesPut(t *testing.T) {
	p := NewKittyProvider()
	s := p.CreateSurface(4, 2)

	cellBuf := make([][]core.Cell, 24)
	for i := range cellBuf {
		cellBuf[i] = make([]core.Cell, 80)
	}
	painter := core.NewPainter(cellBuf, core.Rect{X: 0, Y: 0, W: 80, H: 24})
	s.Place(painter, core.Rect{X: 5, Y: 10, W: 20, H: 8}, -1)

	var buf bytes.Buffer
	if err := p.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "a=p") {
		t.Errorf("expected a=p (put/display), got: %q", output)
	}
	if !strings.Contains(output, "\x1b[11;6H") {
		t.Errorf("expected cursor move to row 11 col 6, got: %q", output)
	}
	if !strings.Contains(output, "c=20") {
		t.Errorf("expected c=20 columns, got: %q", output)
	}
	if !strings.Contains(output, "r=8") {
		t.Errorf("expected r=8 rows, got: %q", output)
	}
}

func TestKittyResetQueuesDeletePlacements(t *testing.T) {
	p := NewKittyProvider()
	p.Reset()

	var buf bytes.Buffer
	if err := p.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "a=d") {
		t.Errorf("expected a=d (delete), got: %q", output)
	}
	if !strings.Contains(output, "d=a") {
		t.Errorf("expected d=a (placements only), got: %q", output)
	}
}

func TestKittyDeleteQueuesFullDelete(t *testing.T) {
	p := NewKittyProvider()
	s := p.CreateSurface(10, 10)
	id := s.ID()

	s.Delete()

	var buf bytes.Buffer
	if err := p.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "d=I") {
		t.Errorf("expected d=I (full delete), got: %q", output)
	}
	if !strings.Contains(output, fmt.Sprintf("i=%d", id)) {
		t.Errorf("expected i=%d, got: %q", id, output)
	}
}

func TestKittyCapability(t *testing.T) {
	p := NewKittyProvider()
	if p.Capability() != core.GraphicsKitty {
		t.Errorf("expected GraphicsKitty, got %d", p.Capability())
	}
}

func TestKittyUpdateChunking(t *testing.T) {
	p := NewKittyProvider()
	// Use varied pixels to defeat PNG compression and exceed 4096 bytes base64
	s := p.CreateSurface(200, 200)
	for y := range 200 {
		for x := range 200 {
			s.Buffer().Set(x, y, color.RGBA{uint8((x * 7 + y * 13) % 256), uint8((x * 11 + y * 3) % 256), uint8((x * 5 + y * 9) % 256), 255})
		}
	}

	if err := s.Update(); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	var buf bytes.Buffer
	if err := p.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "m=1") {
		t.Error("expected m=1 for continuation chunk")
	}
	if !strings.Contains(output, "m=0") {
		t.Error("expected m=0 for final chunk")
	}
}
