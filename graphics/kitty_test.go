package graphics

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestEncodeKittyPlacement(t *testing.T) {
	// Small PNG-like payload (just test encoding, not valid PNG)
	data := []byte("test-image-data")

	var buf bytes.Buffer
	p := &KittyProvider{nextID: 1}
	p.PlaceImage(core.ImagePlacement{
		ID:      1,
		Rect:    core.Rect{X: 5, Y: 10, W: 20, H: 8},
		ImgData: data,
		ZIndex:  -1,
	})

	if len(p.pending) != 1 {
		t.Fatalf("expected 1 pending placement, got %d", len(p.pending))
	}

	err := p.flushTo(&buf)
	if err != nil {
		t.Fatalf("flush error: %v", err)
	}

	output := buf.String()

	// Should contain cursor positioning (1-indexed: row=11, col=6)
	if !strings.Contains(output, "\x1b[11;6H") {
		t.Errorf("expected cursor move to row 11, col 6 (1-indexed), got: %q", output)
	}

	// Should contain APC intro
	if !strings.Contains(output, "\x1b_G") {
		t.Errorf("expected APC graphics intro, got: %q", output)
	}

	// Should contain control keys
	if !strings.Contains(output, "a=T") {
		t.Errorf("expected a=T (transmit+display), got: %q", output)
	}
	if !strings.Contains(output, "f=100") {
		t.Errorf("expected f=100 (PNG format), got: %q", output)
	}
	if !strings.Contains(output, "c=20") {
		t.Errorf("expected c=20 (columns), got: %q", output)
	}
	if !strings.Contains(output, "r=8") {
		t.Errorf("expected r=8 (rows), got: %q", output)
	}
	if !strings.Contains(output, "q=2") {
		t.Errorf("expected q=2 (suppress errors), got: %q", output)
	}

	// Should contain base64-encoded payload
	encoded := base64.StdEncoding.EncodeToString(data)
	if !strings.Contains(output, encoded) {
		t.Errorf("expected base64 payload %q in output", encoded)
	}

	// Should end with ST (string terminator)
	if !strings.HasSuffix(output, "\x1b\\") {
		t.Errorf("expected ST terminator, got: %q", output[len(output)-4:])
	}

	// Pending should be cleared
	if len(p.pending) != 0 {
		t.Errorf("expected pending cleared after flush, got %d", len(p.pending))
	}
}

func TestEncodeKittyChunking(t *testing.T) {
	// Create payload larger than 4096 base64 bytes
	// 4096 base64 chars = 3072 raw bytes, so use 4000 raw bytes
	data := make([]byte, 4000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	p := &KittyProvider{nextID: 1}
	p.PlaceImage(core.ImagePlacement{
		ID:      1,
		Rect:    core.Rect{X: 0, Y: 0, W: 10, H: 5},
		ImgData: data,
	})

	var buf bytes.Buffer
	err := p.flushTo(&buf)
	if err != nil {
		t.Fatalf("flush error: %v", err)
	}

	output := buf.String()

	// Should have multiple APC sequences (m=1 for continuation, m=0 for final)
	if !strings.Contains(output, "m=1") {
		t.Error("expected m=1 for continuation chunk")
	}
	if !strings.Contains(output, "m=0") {
		t.Error("expected m=0 for final chunk")
	}

	// Count APC sequences
	count := strings.Count(output, "\x1b_G")
	if count < 2 {
		t.Errorf("expected at least 2 APC sequences for chunked data, got %d", count)
	}
}

func TestKittyDeleteImage(t *testing.T) {
	var buf bytes.Buffer
	p := &KittyProvider{nextID: 1}

	p.DeleteImage(42)
	err := p.flushTo(&buf)
	if err != nil {
		t.Fatalf("flush error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "a=d") {
		t.Errorf("expected a=d (delete action), got: %q", output)
	}
	if !strings.Contains(output, "d=I") {
		t.Errorf("expected d=I (delete by ID + free), got: %q", output)
	}
	if !strings.Contains(output, "i=42") {
		t.Errorf("expected i=42, got: %q", output)
	}
}

func TestKittyDeleteAll(t *testing.T) {
	var buf bytes.Buffer
	p := &KittyProvider{nextID: 1}

	p.DeleteAll()
	err := p.flushTo(&buf)
	if err != nil {
		t.Fatalf("flush error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "a=d") {
		t.Errorf("expected a=d, got: %q", output)
	}
	if !strings.Contains(output, "d=A") {
		t.Errorf("expected d=A (delete all + free), got: %q", output)
	}
}

func TestKittyCapability(t *testing.T) {
	p := &KittyProvider{}
	if p.Capability() != core.GraphicsKitty {
		t.Errorf("expected GraphicsKitty, got %d", p.Capability())
	}
}
