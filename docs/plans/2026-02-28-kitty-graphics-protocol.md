# Kitty Graphics Protocol Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Kitty graphics protocol support to the Image widget with auto-detection and half-block fallback.

**Architecture:** A `GraphicsProvider` interface in `core/` abstracts image rendering. The standalone runtime detects Kitty support at startup and injects either a `KittyProvider` or `HalfBlockProvider`. The Image widget queries the provider during `Draw()` and renders via Kitty APC sequences when available, falling back to existing half-block art otherwise.

**Tech Stack:** Go 1.24, tcell v2 (for screen/tty access), Kitty graphics protocol (APC escape sequences with base64 PNG payloads).

**Design doc:** `docs/plans/2026-02-28-kitty-graphics-protocol-design.md`

---

### Task 1: Core GraphicsProvider Interface

**Files:**
- Create: `core/graphics.go`
- Test: `core/graphics_test.go`

**Step 1: Write the test**

```go
// core/graphics_test.go
package core

import "testing"

func TestGraphicsCapabilityOrdering(t *testing.T) {
	if GraphicsNone >= GraphicsHalfBlock {
		t.Error("None should be less than HalfBlock")
	}
	if GraphicsHalfBlock >= GraphicsKitty {
		t.Error("HalfBlock should be less than Kitty")
	}
}

func TestImagePlacementFields(t *testing.T) {
	p := ImagePlacement{
		ID:      42,
		Rect:    Rect{X: 5, Y: 10, W: 20, H: 8},
		ImgData: []byte{0x89, 0x50, 0x4E, 0x47},
		ZIndex:  -1,
	}
	if p.ID != 42 {
		t.Errorf("expected ID 42, got %d", p.ID)
	}
	if p.Rect.W != 20 || p.Rect.H != 8 {
		t.Errorf("unexpected rect: %+v", p.Rect)
	}
	if p.ZIndex != -1 {
		t.Errorf("expected ZIndex -1, got %d", p.ZIndex)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/marc/projects/texel/texelui && go test ./core/ -run TestGraphics -v`
Expected: FAIL — `GraphicsNone`, `ImagePlacement` undefined

**Step 3: Write the implementation**

```go
// core/graphics.go
package core

// GraphicsCapability describes what level of image rendering a provider supports.
type GraphicsCapability int

const (
	GraphicsNone      GraphicsCapability = iota
	GraphicsHalfBlock                           // Unicode half-block art (always available)
	GraphicsKitty                               // Kitty graphics protocol (APC sequences)
)

// ImagePlacement represents an image to display at a screen position.
type ImagePlacement struct {
	ID      uint32 // unique image ID for updates/deletion
	Rect    Rect   // screen cell region to occupy
	ImgData []byte // raw PNG/JPEG/GIF bytes
	ZIndex  int    // negative = behind text
}

// GraphicsProvider abstracts image rendering capabilities.
// The runtime injects an implementation at startup.
// Widgets query Capability() to choose their rendering strategy.
type GraphicsProvider interface {
	Capability() GraphicsCapability
	PlaceImage(p ImagePlacement) error
	DeleteImage(id uint32)
	DeleteAll()
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/marc/projects/texel/texelui && go test ./core/ -run TestGraphics -v`
Expected: PASS

**Step 5: Commit**

```bash
git add core/graphics.go core/graphics_test.go
git commit -m "Add GraphicsProvider interface and types"
```

---

### Task 2: Wire GraphicsProvider into Painter

**Files:**
- Modify: `core/painter.go`
- Modify: `core/uimanager.go`
- Test: `core/graphics_test.go` (append)

**Step 1: Write the test**

Append to `core/graphics_test.go`:

```go
func TestPainterGraphicsProvider(t *testing.T) {
	buf := make([][]Cell, 2)
	for i := range buf {
		buf[i] = make([]Cell, 4)
	}

	// Without provider
	p := NewPainter(buf, Rect{X: 0, Y: 0, W: 4, H: 2})
	if p.GraphicsProvider() != nil {
		t.Error("expected nil provider by default")
	}

	// With provider
	p2 := NewPainterWithGraphics(buf, Rect{X: 0, Y: 0, W: 4, H: 2}, &testProvider{cap: GraphicsKitty})
	gp := p2.GraphicsProvider()
	if gp == nil {
		t.Fatal("expected non-nil provider")
	}
	if gp.Capability() != GraphicsKitty {
		t.Errorf("expected Kitty, got %d", gp.Capability())
	}
}

// testProvider is a minimal GraphicsProvider for testing.
type testProvider struct {
	cap        GraphicsCapability
	placements []ImagePlacement
	deleted    []uint32
	allDeleted bool
}

func (p *testProvider) Capability() GraphicsCapability { return p.cap }
func (p *testProvider) PlaceImage(pl ImagePlacement) error {
	p.placements = append(p.placements, pl)
	return nil
}
func (p *testProvider) DeleteImage(id uint32) { p.deleted = append(p.deleted, id) }
func (p *testProvider) DeleteAll()            { p.allDeleted = true }
```

**Step 2: Run test to verify it fails**

Run: `cd /home/marc/projects/texel/texelui && go test ./core/ -run TestPainterGraphics -v`
Expected: FAIL — `NewPainterWithGraphics` undefined

**Step 3: Modify Painter to carry a GraphicsProvider**

In `core/painter.go`, add a `gp` field and a new constructor:

```go
// Add field to Painter struct:
type Painter struct {
	buf  [][]Cell
	clip Rect
	gp   GraphicsProvider
}

// Add new constructor (keep existing NewPainter unchanged for backward compat):
func NewPainterWithGraphics(buf [][]Cell, clip Rect, gp GraphicsProvider) *Painter {
	return &Painter{buf: buf, clip: clip, gp: gp}
}

// Add accessor:
func (p *Painter) GraphicsProvider() GraphicsProvider { return p.gp }
```

Also add `WithGraphics` to `WithClip` so clipped painters inherit the provider:

```go
func (p *Painter) WithClip(rect Rect) *Painter {
	// ... existing intersection logic ...
	return &Painter{
		buf: p.buf,
		clip: Rect{...},
		gp:  p.gp,  // propagate graphics provider
	}
}
```

**Step 4: Modify UIManager.Render() to pass provider to Painter**

In `core/uimanager.go`, add a `graphicsProvider` field to `UIManager`:

```go
// In UIManager struct, add:
graphicsProvider GraphicsProvider

// Add setter:
func (u *UIManager) SetGraphicsProvider(gp GraphicsProvider) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.graphicsProvider = gp
}

func (u *UIManager) GraphicsProvider() GraphicsProvider {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.graphicsProvider
}
```

In `Render()`, change `NewPainter` calls to `NewPainterWithGraphics`:

- Line 874: `p := NewPainterWithGraphics(u.buf, full, u.graphicsProvider)`
- Line 908: `p := NewPainterWithGraphics(u.buf, clip, u.graphicsProvider)`

**Step 5: Run test to verify it passes**

Run: `cd /home/marc/projects/texel/texelui && go test ./core/ -v`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add core/painter.go core/uimanager.go core/graphics_test.go
git commit -m "Wire GraphicsProvider through Painter and UIManager"
```

---

### Task 3: HalfBlockProvider

**Files:**
- Create: `graphics/halfblock.go`
- Test: `graphics/halfblock_test.go`

**Step 1: Write the test**

```go
// graphics/halfblock_test.go
package graphics

import (
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestHalfBlockProvider(t *testing.T) {
	p := NewHalfBlockProvider()
	if p.Capability() != core.GraphicsHalfBlock {
		t.Errorf("expected HalfBlock, got %d", p.Capability())
	}
	// PlaceImage is a no-op, should not error
	err := p.PlaceImage(core.ImagePlacement{ID: 1})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Delete methods are no-ops, should not panic
	p.DeleteImage(1)
	p.DeleteAll()
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/marc/projects/texel/texelui && go test ./graphics/ -run TestHalfBlock -v`
Expected: FAIL — package doesn't exist

**Step 3: Write the implementation**

```go
// graphics/halfblock.go
package graphics

import "github.com/framegrace/texelui/core"

// HalfBlockProvider signals that only Unicode half-block art is available.
// All image operations are no-ops; the Image widget handles rendering itself.
type HalfBlockProvider struct{}

func NewHalfBlockProvider() *HalfBlockProvider    { return &HalfBlockProvider{} }
func (p *HalfBlockProvider) Capability() core.GraphicsCapability { return core.GraphicsHalfBlock }
func (p *HalfBlockProvider) PlaceImage(core.ImagePlacement) error { return nil }
func (p *HalfBlockProvider) DeleteImage(uint32)   {}
func (p *HalfBlockProvider) DeleteAll()            {}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/marc/projects/texel/texelui && go test ./graphics/ -run TestHalfBlock -v`
Expected: PASS

**Step 5: Commit**

```bash
git add graphics/halfblock.go graphics/halfblock_test.go
git commit -m "Add HalfBlockProvider (no-op fallback)"
```

---

### Task 4: Kitty Protocol Encoding

**Files:**
- Create: `graphics/kitty.go`
- Test: `graphics/kitty_test.go`

**Step 1: Write the test for APC sequence encoding**

```go
// graphics/kitty_test.go
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

	// Should contain cursor positioning
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
	// Should contain delete command with image ID
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
```

**Step 2: Run test to verify it fails**

Run: `cd /home/marc/projects/texel/texelui && go test ./graphics/ -run TestKitty -v`
Expected: FAIL — `KittyProvider` undefined

**Step 3: Write the KittyProvider implementation**

```go
// graphics/kitty.go
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

func NewKittyProvider() *KittyProvider {
	return &KittyProvider{nextID: 1}
}

func (p *KittyProvider) Capability() core.GraphicsCapability {
	return core.GraphicsKitty
}

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

func (p *KittyProvider) DeleteImage(id uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(p.pending, kittyCommand{deleteID: id})
}

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
```

**Step 4: Run test to verify it passes**

Run: `cd /home/marc/projects/texel/texelui && go test ./graphics/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add graphics/kitty.go graphics/kitty_test.go
git commit -m "Add KittyProvider with APC encoding and chunking"
```

---

### Task 5: Kitty Capability Detection

**Files:**
- Create: `graphics/detect.go`
- Test: `graphics/detect_test.go`

**Step 1: Write the test**

```go
// graphics/detect_test.go
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
	// Terminal responds with OK for our query image ID
	tty := &mockTTY{
		response: []byte("\x1b_Gi=31;OK\x1b\\"),
	}
	cap := detectFromReadWriter(tty, tty)
	if cap != core.GraphicsKitty {
		t.Errorf("expected GraphicsKitty, got %d", cap)
	}
}

func TestDetectKittyUnsupported(t *testing.T) {
	// Terminal responds with DA only (no graphics response)
	tty := &mockTTY{
		response: []byte("\x1b[?62;c"),
	}
	cap := detectFromReadWriter(tty, tty)
	if cap != core.GraphicsHalfBlock {
		t.Errorf("expected GraphicsHalfBlock fallback, got %d", cap)
	}
}

func TestDetectKittyEmpty(t *testing.T) {
	// No response at all
	tty := &mockTTY{
		response: []byte{},
	}
	cap := detectFromReadWriter(tty, tty)
	if cap != core.GraphicsHalfBlock {
		t.Errorf("expected GraphicsHalfBlock fallback, got %d", cap)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/marc/projects/texel/texelui && go test ./graphics/ -run TestDetect -v`
Expected: FAIL — `detectFromReadWriter` undefined

**Step 3: Write the detection implementation**

```go
// graphics/detect.go
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

	// Read response with timeout via goroutine
	responseCh := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 256)
		var collected []byte
		for {
			n, err := r.Read(buf)
			if n > 0 {
				collected = append(collected, buf[:n]...)
				// Check if we have enough to determine the result
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
```

**Step 4: Run test to verify it passes**

Run: `cd /home/marc/projects/texel/texelui && go test ./graphics/ -run TestDetect -v`
Expected: PASS

**Step 5: Commit**

```bash
git add graphics/detect.go graphics/detect_test.go
git commit -m "Add Kitty graphics capability detection"
```

---

### Task 6: Refactor Image Widget for Dual Render Path

**Files:**
- Modify: `widgets/image.go`
- Modify: `widgets/image_test.go`

**Step 1: Write the test for Kitty render path**

Append to `widgets/image_test.go`:

```go
func TestImage_KittyPath(t *testing.T) {
	// Create a small test image
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	pngData := buf.Bytes()

	imgWidget := NewImage(pngData, "red square")
	imgWidget.Resize(4, 2)
	imgWidget.SetPosition(5, 10)

	// Create a provider that records placements
	provider := &mockGraphicsProvider{cap: core.GraphicsKitty}

	cellBuf := createTestBuffer(20, 15)
	p := core.NewPainterWithGraphics(cellBuf, core.Rect{X: 0, Y: 0, W: 20, H: 15}, provider)

	imgWidget.Draw(p)

	// Should have called PlaceImage
	if len(provider.placements) != 1 {
		t.Fatalf("expected 1 placement, got %d", len(provider.placements))
	}

	pl := provider.placements[0]
	if pl.Rect.X != 5 || pl.Rect.Y != 10 {
		t.Errorf("expected position (5,10), got (%d,%d)", pl.Rect.X, pl.Rect.Y)
	}
	if pl.Rect.W != 4 || pl.Rect.H != 2 {
		t.Errorf("expected size (4,2), got (%d,%d)", pl.Rect.W, pl.Rect.H)
	}
	if len(pl.ImgData) == 0 {
		t.Error("expected non-empty image data")
	}

	// Cell buffer should be filled with spaces (placeholder)
	for y := 10; y < 12; y++ {
		for x := 5; x < 9; x++ {
			if cellBuf[y][x].Ch != ' ' {
				t.Errorf("expected space at (%d,%d), got %q", x, y, cellBuf[y][x].Ch)
			}
		}
	}
}

func TestImage_FallbackToHalfBlock(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{0, 255, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	imgWidget := NewImage(buf.Bytes(), "green square")
	imgWidget.Resize(4, 2)

	// HalfBlock provider — should use half-block rendering
	provider := &mockGraphicsProvider{cap: core.GraphicsHalfBlock}
	cellBuf := createTestBuffer(4, 2)
	p := core.NewPainterWithGraphics(cellBuf, core.Rect{X: 0, Y: 0, W: 4, H: 2}, provider)

	imgWidget.Draw(p)

	// Should NOT have called PlaceImage
	if len(provider.placements) != 0 {
		t.Errorf("expected 0 placements for HalfBlock, got %d", len(provider.placements))
	}

	// Should have rendered half-blocks
	if cellBuf[0][0].Ch != '\u2580' {
		t.Errorf("expected half-block character, got %q", cellBuf[0][0].Ch)
	}
}

// mockGraphicsProvider for widget tests
type mockGraphicsProvider struct {
	cap        core.GraphicsCapability
	placements []core.ImagePlacement
}

func (p *mockGraphicsProvider) Capability() core.GraphicsCapability { return p.cap }
func (p *mockGraphicsProvider) PlaceImage(pl core.ImagePlacement) error {
	p.placements = append(p.placements, pl)
	return nil
}
func (p *mockGraphicsProvider) DeleteImage(uint32) {}
func (p *mockGraphicsProvider) DeleteAll()          {}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/marc/projects/texel/texelui && go test ./widgets/ -run "TestImage_Kitty|TestImage_Fallback" -v`
Expected: FAIL — `NewPainterWithGraphics` exists but Image widget doesn't use it yet

**Step 3: Modify Image widget**

Changes to `widgets/image.go`:
1. Keep raw PNG bytes in a `pngBytes` field (don't discard after decode)
2. Add `imageID uint32` field
3. Add `lastRect Rect` for move detection
4. Extract current Draw logic into `drawHalfBlock(p)`
5. New `Draw()` checks provider and branches

```go
type Image struct {
	core.BaseWidget
	imgData  []byte      // kept nil after decode (legacy)
	pngBytes []byte      // raw PNG bytes retained for Kitty f=100
	altText  string
	decoded  image.Image
	valid    bool
	style    tcell.Style
	imageID  uint32      // Kitty image ID (0 = not yet allocated)
	lastRect core.Rect   // last placed rect for move detection

	inv func(core.Rect)
}

func NewImage(imgData []byte, altText string) *Image {
	// ... existing setup ...
	// Keep a copy of raw bytes for Kitty:
	img.pngBytes = make([]byte, len(imgData))
	copy(img.pngBytes, imgData)
	// ... decode as before ...
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
	if gp != nil && gp.Capability() >= core.GraphicsKitty {
		img.drawKitty(p, gp)
		return
	}

	img.drawHalfBlock(p)
}

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
		ImgData: img.pngBytes,
		ZIndex:  -1,
	})
	img.lastRect = img.Rect
}

func (img *Image) drawHalfBlock(p *core.Painter) {
	// ... existing Draw logic (half-block rendering) moved here ...
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /home/marc/projects/texel/texelui && go test ./widgets/ -v`
Expected: ALL PASS (both new Kitty tests and existing half-block tests)

**Step 5: Commit**

```bash
git add widgets/image.go widgets/image_test.go
git commit -m "Add Kitty render path to Image widget with fallback"
```

---

### Task 7: Runtime Integration

**Files:**
- Modify: `runtime/runner.go`

**Step 1: Write the test**

The runtime detection and flush involve real terminal I/O that's hard to unit test. Instead, verify the wiring is correct by checking the integration compiles and the draw loop structure is sound.

This task is integration-only — no new test file. The existing tests in `graphics/` cover the protocol encoding and detection logic.

Verify existing tests still pass after changes:

Run: `cd /home/marc/projects/texel/texelui && go test ./...`
Expected: ALL PASS

**Step 2: Modify `runtime/runner.go`**

Add import for `graphics` package. Modify `runApp()`:

After `screen.Init()` and before `app.Resize()`:

```go
import "github.com/framegrace/texelui/graphics"

// In runApp(), after screen.Init():

// Detect graphics capability
var graphicsProvider core.GraphicsProvider
if tty, ok := screen.Tty(); ok {
	cap := graphics.DetectCapability(tty)
	if cap == core.GraphicsKitty {
		graphicsProvider = graphics.NewKittyProvider()
	} else {
		graphicsProvider = graphics.NewHalfBlockProvider()
	}
} else {
	graphicsProvider = graphics.NewHalfBlockProvider()
}

// Inject into UIManager if the app supports it
if ua, ok := app.(interface{ UIManager() *core.UIManager }); ok {
	ua.UIManager().SetGraphicsProvider(graphicsProvider)
}

// Cleanup on exit (add to defer chain after screen.Fini):
defer graphicsProvider.DeleteAll()
```

Modify the `draw` closure to flush Kitty commands after `screen.Show()`:

```go
draw := func() {
	screen.Clear()
	buffer := app.Render()
	if buffer != nil {
		for y := 0; y < len(buffer); y++ {
			row := buffer[y]
			for x := 0; x < len(row); x++ {
				cell := row[x]
				screen.SetContent(x, y, cell.Ch, nil, cell.Style)
			}
		}
	}
	screen.Show()

	// Flush queued Kitty image commands after tcell has flushed
	if kp, ok := graphicsProvider.(*graphics.KittyProvider); ok {
		if tty, hasTty := screen.Tty(); hasTty {
			_ = kp.Flush(tty)
		}
	}
}
```

In the `EventResize` handler, clear all images before redraw:

```go
case *tcell.EventResize:
	w, h := tev.Size()
	graphicsProvider.DeleteAll()
	app.Resize(w, h)
	draw()
```

**Step 3: Verify the adapter exposes UIManager**

Check `adapter/texel_app.go` — `UIApp` likely already has a `UIManager()` accessor. If not, add one.

Run: `cd /home/marc/projects/texel/texelui && grep -n "UIManager" adapter/texel_app.go`

If `UIManager()` method doesn't exist, add to `adapter/texel_app.go`:

```go
func (a *UIApp) UIManager() *core.UIManager { return a.ui }
```

**Step 4: Build and run tests**

Run: `cd /home/marc/projects/texel/texelui && go build ./... && go test ./...`
Expected: ALL PASS, no build errors

**Step 5: Commit**

```bash
git add runtime/runner.go adapter/texel_app.go
git commit -m "Integrate Kitty graphics detection and flush into runtime"
```

---

### Task 8: Manual Verification

**Step 1: Build the demo**

Run: `cd /home/marc/projects/texel/texelui && make demos`

**Step 2: Test in a Kitty-compatible terminal**

Run the demo in a terminal that supports the Kitty graphics protocol (Kitty, WezTerm, Ghostty):

```bash
./bin/texelui-demo
```

Navigate to the Widgets tab. The gradient image should render as a true raster image (smooth gradient) instead of the blocky half-block art.

**Step 3: Test in a non-Kitty terminal**

Run the same demo in a basic terminal (e.g., xterm, plain gnome-terminal):

```bash
./bin/texelui-demo
```

The gradient should render as half-block art (the existing behavior), with no error messages or visual glitches from the Kitty detection probe.

**Step 4: Commit (if any fixes needed)**

Fix any issues found during manual testing and commit.

---

### Summary

| Task | What | Files |
|------|------|-------|
| 1 | Core `GraphicsProvider` interface | `core/graphics.go`, `core/graphics_test.go` |
| 2 | Wire provider into `Painter` + `UIManager` | `core/painter.go`, `core/uimanager.go` |
| 3 | `HalfBlockProvider` (no-op fallback) | `graphics/halfblock.go`, `graphics/halfblock_test.go` |
| 4 | `KittyProvider` (APC encoding + chunking) | `graphics/kitty.go`, `graphics/kitty_test.go` |
| 5 | Kitty capability detection | `graphics/detect.go`, `graphics/detect_test.go` |
| 6 | Image widget dual render path | `widgets/image.go`, `widgets/image_test.go` |
| 7 | Runtime integration (detection + flush) | `runtime/runner.go`, `adapter/texel_app.go` |
| 8 | Manual verification | — |
