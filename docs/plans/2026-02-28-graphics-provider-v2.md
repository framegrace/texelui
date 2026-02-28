# GraphicsProvider v2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign GraphicsProvider with upload/place separation via ImageSurface, working in both standalone and texelation modes.

**Architecture:** Replace `PlaceImage(ImagePlacement)` with `ImageSurface` objects that separate `Update()` (upload) from `Place()` (display). Three providers: `KittyProvider`, `HalfBlockProvider`, `RemoteGraphicsProvider`. Four new texelation protocol messages. Client-side image cache and renderer integration.

**Tech Stack:** Go 1.24, tcell/v2, texelui core, texelation protocol (binary LE encoding)

---

## Phase 1: Core API + Standalone (texelui)

### Task 1: Update core interfaces

**Files:**
- Modify: `core/graphics.go`
- Modify: `core/graphics_test.go`

**Step 1: Write failing tests**

Add to `core/graphics_test.go`:

```go
import "image"

func TestImageSurfaceInterface(t *testing.T) {
	// Verify ImageSurface has the required methods
	var _ ImageSurface = (*mockSurface)(nil)
}

type mockSurface struct {
	id  uint32
	buf *image.RGBA
}

func (m *mockSurface) ID() uint32                                { return m.id }
func (m *mockSurface) Buffer() *image.RGBA                       { return m.buf }
func (m *mockSurface) Update() error                             { return nil }
func (m *mockSurface) Place(p *Painter, rect Rect, zIndex int)   {}
func (m *mockSurface) Delete()                                   {}

func TestGraphicsProviderV2Interface(t *testing.T) {
	var _ GraphicsProvider = (*mockProviderV2)(nil)
}

type mockProviderV2 struct{}

func (m *mockProviderV2) Capability() GraphicsCapability         { return GraphicsKitty }
func (m *mockProviderV2) CreateSurface(w, h int) ImageSurface    { return nil }
func (m *mockProviderV2) Reset()                                 {}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./core/ -run TestImageSurface -v`
Expected: FAIL — `ImageSurface` not defined

**Step 3: Update core/graphics.go**

Replace the entire file with the v2 interfaces:

```go
package core

import "image"

// GraphicsCapability describes what level of image rendering a provider supports.
type GraphicsCapability int

const (
	GraphicsNone      GraphicsCapability = iota
	GraphicsHalfBlock                    // Unicode half-block art (always available)
	GraphicsKitty                        // Kitty graphics protocol (APC sequences)
)

// GraphicsProvider abstracts image rendering capabilities.
// The runtime injects an implementation at startup.
// Widgets query Capability() to choose their rendering strategy.
type GraphicsProvider interface {
	Capability() GraphicsCapability
	CreateSurface(width, height int) ImageSurface
	Reset() // clear all active placements (cached data preserved)
}

// ImageSurface represents an allocated image buffer that can be rendered
// into and displayed at a screen position. The app renders into Buffer(),
// calls Update() when content changes, and calls Place() every frame.
type ImageSurface interface {
	ID() uint32
	Buffer() *image.RGBA
	Update() error
	Place(p *Painter, rect Rect, zIndex int)
	Delete()
}
```

Note: Remove the old `ImagePlacement` struct and the old `GraphicsProvider` interface. The `AllocateID()` method is no longer needed — `CreateSurface` handles ID allocation internally.

**Step 4: Fix compilation errors in existing code**

The old `PlaceImage`, `DeleteImage`, `DeleteAll`, `AllocateID` references will break. Comment out or stub the broken callers temporarily (they'll be rewritten in subsequent tasks). Files affected:
- `graphics/kitty.go` — will be rewritten in Task 3
- `graphics/halfblock.go` — will be rewritten in Task 4
- `widgets/image.go` — will be rewritten in Task 5
- `runtime/runner.go` — will be updated in Task 6

For now, make the minimum changes to compile:
- In `graphics/kitty.go`: add stub methods `CreateSurface`, `Reset` to `KittyProvider`, remove `PlaceImage`, `DeleteImage`, `DeleteAll`, `AllocateID`. Keep `Flush` and internal APC encoding.
- In `graphics/halfblock.go`: add stub methods, remove old ones.
- In `widgets/image.go`: comment out `drawKitty` body, remove references to old API.
- In `runtime/runner.go`: replace `graphicsProvider.DeleteAll()` with `graphicsProvider.Reset()`, remove old `PlaceImage`-related code.

**Step 5: Run tests to verify they pass**

Run: `go build ./... && go test ./core/ -v`
Expected: PASS — all core tests pass, project compiles

**Step 6: Commit**

```bash
git add core/graphics.go core/graphics_test.go graphics/ widgets/image.go runtime/runner.go
git commit -m "Replace GraphicsProvider with v2 API: ImageSurface + Reset"
```

---

### Task 2: Rewrite KittyProvider with surface support

**Files:**
- Modify: `graphics/kitty.go`
- Modify: `graphics/kitty_test.go`

**Step 1: Write failing tests**

Replace `graphics/kitty_test.go` with tests for the new surface API:

```go
package graphics

import (
	"bytes"
	"image"
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

	// Draw a pixel so the image has content
	s.Buffer().Set(0, 0, color.RGBA{255, 0, 0, 255})

	if err := s.Update(); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	var buf bytes.Buffer
	if err := p.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	output := buf.String()
	// Should contain transmit action (a=t), not display (a=T or a=p)
	if !strings.Contains(output, "a=t") {
		t.Errorf("expected a=t (transmit only), got: %q", output)
	}
	// Should contain PNG format
	if !strings.Contains(output, "f=100") {
		t.Errorf("expected f=100 (PNG), got: %q", output)
	}
	// Should contain the surface ID
	if !strings.Contains(output, "i=1") {
		t.Errorf("expected i=1, got: %q", output)
	}
}

func TestKittyPlaceQueuesPut(t *testing.T) {
	p := NewKittyProvider()
	s := p.CreateSurface(4, 2)

	painter := core.NewPainter(make([][]core.Cell, 10), core.Rect{X: 0, Y: 0, W: 80, H: 24})
	s.Place(painter, core.Rect{X: 5, Y: 10, W: 20, H: 8}, -1)

	var buf bytes.Buffer
	if err := p.Flush(&buf); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	output := buf.String()
	// Should contain put action (a=p)
	if !strings.Contains(output, "a=p") {
		t.Errorf("expected a=p (put/display), got: %q", output)
	}
	// Should contain cursor positioning (1-indexed: row=11, col=6)
	if !strings.Contains(output, "\x1b[11;6H") {
		t.Errorf("expected cursor move, got: %q", output)
	}
	// Should contain cell dimensions
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
	// d=a means delete all placements (keep data)
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
	// d=I means delete data + placements for specific ID
	if !strings.Contains(output, "d=I") {
		t.Errorf("expected d=I (full delete), got: %q", output)
	}
	if !strings.Contains(output, "i="+fmt.Sprint(id)) {
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
	// Create large enough image to require chunking (>4096 base64 bytes)
	s := p.CreateSurface(100, 100)
	// Fill with non-trivial data so PNG is large
	for y := range 100 {
		for x := range 100 {
			s.Buffer().Set(x, y, color.RGBA{uint8(x), uint8(y), 128, 255})
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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./graphics/ -run TestKitty -v`
Expected: FAIL — stub methods don't implement surface API

**Step 3: Rewrite kitty.go**

Replace `graphics/kitty.go` with the new surface-based implementation:

```go
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
)

const maxChunkSize = 4096 // max base64 bytes per APC sequence

// kittyCommand represents a queued Kitty protocol command.
type kittyCommand struct {
	cmdType  kittyCommandType
	id       uint32
	rect     core.Rect
	zIndex   int
	pngData  []byte // only for transmit commands
}

type kittyCommandType int

const (
	cmdTransmit       kittyCommandType = iota // a=t (upload, no display)
	cmdPut                                     // a=p (display from cache)
	cmdDeleteID                                // a=d,d=I (delete data+placements by ID)
	cmdResetPlacements                         // a=d,d=a (delete all placements, keep data)
)

// kittySurface implements core.ImageSurface for the Kitty provider.
type kittySurface struct {
	provider *KittyProvider
	id       uint32
	buf      *image.RGBA
}

func (s *kittySurface) ID() uint32         { return s.id }
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
	// Fill region with spaces so tcell clears the area
	p.Fill(rect, ' ', p.DefaultStyle())
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

// NewKittyProvider creates a KittyProvider with IDs starting at 1.
func NewKittyProvider() *KittyProvider {
	return &KittyProvider{nextID: 1}
}

// Capability returns GraphicsKitty.
func (p *KittyProvider) Capability() core.GraphicsCapability {
	return core.GraphicsKitty
}

// CreateSurface allocates an image buffer and returns a surface.
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

// Reset clears all active placements (cached image data is preserved).
func (p *KittyProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(p.pending, kittyCommand{cmdType: cmdResetPlacements})
}

// Flush writes all pending commands to w, then clears the queue.
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

// writeTransmit sends image data without displaying (a=t).
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

// writePut displays an already-transmitted image (a=p).
func writePut(w io.Writer, id uint32, rect core.Rect, zIndex int) error {
	// Move cursor to placement position (1-indexed)
	if _, err := fmt.Fprintf(w, "\x1b[%d;%dH", rect.Y+1, rect.X+1); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w,
		"\x1b_Ga=p,i=%d,c=%d,r=%d,z=%d,q=2;\x1b\\",
		id, rect.W, rect.H, zIndex)
	return err
}

// writeDeleteByID deletes image data and all placements for a specific ID.
func writeDeleteByID(w io.Writer, id uint32) error {
	_, err := fmt.Fprintf(w, "\x1b_Ga=d,d=I,i=%d,q=2;\x1b\\\n", id)
	return err
}

// writeResetPlacements deletes all visible placements but keeps cached data.
func writeResetPlacements(w io.Writer) error {
	_, err := fmt.Fprintf(w, "\x1b_Ga=d,d=a,q=2;\x1b\\")
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

**Important**: The `Place` method calls `p.Fill(rect, ' ', p.DefaultStyle())`. The `Painter` needs a `DefaultStyle()` accessor. Check if it exists; if not, use `tcell.StyleDefault` directly or add the accessor in Task 1.

If `Painter` doesn't have `DefaultStyle()`, use this instead in `Place`:
```go
p.Fill(rect, ' ', tcell.StyleDefault)
```

**Step 4: Run tests**

Run: `go test ./graphics/ -run TestKitty -v`
Expected: PASS

**Step 5: Commit**

```bash
git add graphics/kitty.go graphics/kitty_test.go
git commit -m "Rewrite KittyProvider with surface-based API"
```

---

### Task 3: Rewrite HalfBlockProvider with surface support

**Files:**
- Modify: `graphics/halfblock.go`
- Modify: `graphics/halfblock_test.go`

**Step 1: Write failing tests**

```go
package graphics

import (
	"image"
	"image/color"
	"testing"

	"github.com/framegrace/texelui/core"
)

func TestHalfBlockCreateSurface(t *testing.T) {
	p := NewHalfBlockProvider()
	s := p.CreateSurface(20, 10)
	if s == nil {
		t.Fatal("CreateSurface returned nil")
	}
	if s.ID() == 0 {
		t.Error("expected non-zero ID")
	}
	buf := s.Buffer()
	if buf.Bounds().Dx() != 20 || buf.Bounds().Dy() != 10 {
		t.Errorf("got %dx%d, want 20x10", buf.Bounds().Dx(), buf.Bounds().Dy())
	}
}

func TestHalfBlockCapability(t *testing.T) {
	p := NewHalfBlockProvider()
	if p.Capability() != core.GraphicsHalfBlock {
		t.Errorf("got %d, want GraphicsHalfBlock", p.Capability())
	}
}

func TestHalfBlockPlaceRendersCharacters(t *testing.T) {
	p := NewHalfBlockProvider()
	s := p.CreateSurface(2, 4) // 2 pixels wide, 4 pixels tall → 2 cols x 2 rows

	// Set pixels: top-left red, bottom-left green
	s.Buffer().Set(0, 0, color.RGBA{255, 0, 0, 255})
	s.Buffer().Set(0, 1, color.RGBA{0, 255, 0, 255})

	buf := make([][]core.Cell, 24)
	for i := range buf {
		buf[i] = make([]core.Cell, 80)
	}
	painter := core.NewPainter(buf, core.Rect{X: 0, Y: 0, W: 80, H: 24})

	s.Place(painter, core.Rect{X: 5, Y: 10, W: 2, H: 2}, 0)

	// Should have upper-half-block at position (5, 10)
	cell := buf[10][5]
	if cell.Ch != '\u2580' {
		t.Errorf("expected upper-half-block, got %q", cell.Ch)
	}
}

func TestHalfBlockUpdateIsNoop(t *testing.T) {
	p := NewHalfBlockProvider()
	s := p.CreateSurface(10, 10)
	// Update should not error (it's a no-op for half-block)
	if err := s.Update(); err != nil {
		t.Errorf("Update returned error: %v", err)
	}
}

func TestHalfBlockResetIsNoop(t *testing.T) {
	p := NewHalfBlockProvider()
	// Should not panic
	p.Reset()
}

func TestHalfBlockDeleteFreesBuffer(t *testing.T) {
	p := NewHalfBlockProvider()
	s := p.CreateSurface(10, 10)
	s.Delete()
	// Buffer should be nil after delete
	if s.Buffer() != nil {
		t.Error("expected nil buffer after Delete")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./graphics/ -run TestHalfBlock -v`
Expected: FAIL

**Step 3: Rewrite halfblock.go**

```go
package graphics

import (
	"image"
	"sync"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

// halfBlockSurface implements core.ImageSurface for the half-block provider.
type halfBlockSurface struct {
	id  uint32
	buf *image.RGBA
}

func (s *halfBlockSurface) ID() uint32          { return s.id }
func (s *halfBlockSurface) Buffer() *image.RGBA { return s.buf }
func (s *halfBlockSurface) Update() error       { return nil }

func (s *halfBlockSurface) Place(p *core.Painter, rect core.Rect, zIndex int) {
	if s.buf == nil {
		return
	}
	imgBounds := s.buf.Bounds()
	imgW := imgBounds.Dx()
	imgH := imgBounds.Dy()

	pixW := rect.W
	pixH := rect.H * 2

	for cy := range rect.H {
		for cx := range rect.W {
			topPixY := cy * 2
			botPixY := cy*2 + 1

			topColor := sampleRGBA(s.buf, cx, topPixY, pixW, pixH, imgW, imgH)
			botColor := sampleRGBA(s.buf, cx, botPixY, pixW, pixH, imgW, imgH)

			style := tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(
					int32(topColor.R), int32(topColor.G), int32(topColor.B))).
				Background(tcell.NewRGBColor(
					int32(botColor.R), int32(botColor.G), int32(botColor.B)))

			p.SetCell(rect.X+cx, rect.Y+cy, '\u2580', style)
		}
	}
}

func (s *halfBlockSurface) Delete() {
	s.buf = nil
}

// sampleRGBA maps output pixel coordinates to source image using nearest-neighbor.
func sampleRGBA(img *image.RGBA, cx, py, pixW, pixH, imgW, imgH int) image.RGBAColor {
	imgX := cx * imgW / pixW
	imgY := py * imgH / pixH
	if imgX >= imgW {
		imgX = imgW - 1
	}
	if imgY >= imgH {
		imgY = imgH - 1
	}
	r, g, b, _ := img.At(imgX, imgY).RGBA()
	return image.RGBAColor{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
}

// HalfBlockProvider renders images using Unicode half-block characters.
type HalfBlockProvider struct {
	mu     sync.Mutex
	nextID uint32
}

// NewHalfBlockProvider creates a HalfBlockProvider.
func NewHalfBlockProvider() *HalfBlockProvider {
	return &HalfBlockProvider{nextID: 1}
}

// Capability returns GraphicsHalfBlock.
func (p *HalfBlockProvider) Capability() core.GraphicsCapability {
	return core.GraphicsHalfBlock
}

// CreateSurface allocates an image buffer.
func (p *HalfBlockProvider) CreateSurface(width, height int) core.ImageSurface {
	p.mu.Lock()
	defer p.mu.Unlock()
	id := p.nextID
	p.nextID++
	return &halfBlockSurface{
		id:  id,
		buf: image.NewRGBA(image.Rect(0, 0, width, height)),
	}
}

// Reset is a no-op for half-block (cell-based rendering is cleared by screen.Clear).
func (p *HalfBlockProvider) Reset() {}
```

**Note on `image.RGBAColor`**: This type doesn't exist in stdlib. Use a local struct instead:

```go
type rgb struct{ R, G, B uint8 }
```

And change `sampleRGBA` return type to `rgb`.

**Step 4: Run tests**

Run: `go test ./graphics/ -run TestHalfBlock -v`
Expected: PASS

**Step 5: Commit**

```bash
git add graphics/halfblock.go graphics/halfblock_test.go
git commit -m "Rewrite HalfBlockProvider with surface-based API"
```

---

### Task 4: Update Image widget to use surfaces

**Files:**
- Modify: `widgets/image.go`
- Modify: `widgets/image_test.go` (if exists, check first)

**Step 1: Write failing test**

```go
func TestImageDrawWithSurface(t *testing.T) {
	// Create a 2x2 red PNG
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := range 2 {
		for x := range 2 {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)

	widget := NewImage(pngBuf.Bytes(), "test")
	widget.SetBounds(0, 0, 4, 2)

	provider := NewHalfBlockProvider() // from graphics package
	buf := make([][]core.Cell, 24)
	for i := range buf {
		buf[i] = make([]core.Cell, 80)
	}
	painter := core.NewPainterWithGraphics(buf, core.Rect{X: 0, Y: 0, W: 80, H: 24}, provider)

	widget.Draw(painter)

	// Should have drawn half-block characters
	cell := buf[0][0]
	if cell.Ch != '\u2580' {
		t.Errorf("expected half-block, got %q", cell.Ch)
	}
}
```

**Step 2: Rewrite image.go to use surfaces**

The Image widget needs to:
1. Store raw image bytes for decoding (keep current behavior for size info)
2. Create a surface on first `Draw()` when a `GraphicsProvider` is available
3. Upload pixel data via `Update()` on first draw (or when image changes)
4. Call `Place()` every frame

```go
package widgets

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"

	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

// Image renders an image using the best available graphics method.
type Image struct {
	core.BaseWidget
	decoded  image.Image
	valid    bool
	style    tcell.Style
	altText  string
	surface  core.ImageSurface
	surfaceW int // surface pixel dimensions (to detect resize)
	surfaceH int
	uploaded bool // whether surface content has been uploaded
}

// NewImage creates an Image widget from raw image bytes (PNG, JPEG, or GIF).
func NewImage(imgData []byte, altText string) *Image {
	tm := theme.Get()
	fg := tm.GetSemanticColor("text.primary")
	bg := tm.GetSemanticColor("bg.surface")

	img := &Image{
		altText: altText,
		style:   tcell.StyleDefault.Foreground(fg).Background(bg),
	}
	img.SetFocusable(false)

	decoded, _, err := image.Decode(bytes.NewReader(imgData))
	if err == nil {
		img.decoded = decoded
		img.valid = true
	}
	return img
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
	if gp == nil || gp.Capability() < core.GraphicsHalfBlock {
		img.drawAltText(p)
		return
	}

	img.ensureSurface(gp)
	img.surface.Place(p, img.Rect, -1)
}

// ensureSurface creates or recreates the surface when needed.
func (img *Image) ensureSurface(gp core.GraphicsProvider) {
	needsNew := img.surface == nil ||
		img.surfaceW != img.decoded.Bounds().Dx() ||
		img.surfaceH != img.decoded.Bounds().Dy()

	if needsNew {
		if img.surface != nil {
			img.surface.Delete()
		}
		bounds := img.decoded.Bounds()
		img.surfaceW = bounds.Dx()
		img.surfaceH = bounds.Dy()
		img.surface = gp.CreateSurface(img.surfaceW, img.surfaceH)
		img.uploaded = false
	}

	if !img.uploaded {
		// Copy decoded image into surface buffer
		buf := img.surface.Buffer()
		for y := range img.surfaceH {
			for x := range img.surfaceW {
				buf.Set(x, y, img.decoded.At(img.decoded.Bounds().Min.X+x, img.decoded.Bounds().Min.Y+y))
			}
		}
		img.surface.Update()
		img.uploaded = true
	}
}

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
```

Note: The half-block rendering logic is now inside `HalfBlockProvider.Place()`, not in the Image widget. The widget just calls `surface.Place()` and the provider handles how to render it.

**Step 3: Run tests**

Run: `go build ./... && go test ./widgets/ -run TestImage -v`
Expected: PASS

**Step 4: Commit**

```bash
git add widgets/image.go widgets/image_test.go
git commit -m "Update Image widget to use ImageSurface API"
```

---

### Task 5: Update runtime runner

**Files:**
- Modify: `runtime/runner.go`

**Step 1: Update draw loop**

The runner needs these changes:
1. Replace `graphicsProvider.DeleteAll()` with `graphicsProvider.Reset()`
2. Keep the `Flush` call after `screen.Show()`
3. Update cleanup defer to use `Reset` + `Flush` (clears placements) and surface-level `Delete` isn't needed at shutdown — `Reset` + `Flush` is sufficient to clear the terminal
4. Remove the resize `DeleteAll()` — `Reset()` is called every frame anyway

In the draw closure:
```go
draw := func() {
	screen.Clear()
	graphicsProvider.Reset()
	buffer := app.Render()
	// ... write cells to screen ...
	screen.Show()
	if kp, ok := graphicsProvider.(*graphics.KittyProvider); ok {
		if tty, hasTty := screen.Tty(); hasTty {
			_ = kp.Flush(tty)
		}
	}
}
```

In the cleanup defer:
```go
defer func() {
	if kp, ok := graphicsProvider.(*graphics.KittyProvider); ok {
		kp.Reset()
		if tty, hasTty := screen.Tty(); hasTty {
			_ = kp.Flush(tty)
		}
	}
}()
```

In the resize handler, remove the `graphicsProvider.DeleteAll()` call. `Reset()` in the draw loop handles it.

**Step 2: Verify build and tests**

Run: `go build ./... && go test ./... -v`
Expected: PASS — everything compiles and tests pass

**Step 3: Commit**

```bash
git add runtime/runner.go
git commit -m "Update runtime runner for GraphicsProvider v2 Reset/Flush"
```

---

### Task 6: Update demo app

**Files:**
- Modify: `apps/texelui-demo/demo.go`

**Step 1: Verify demo compiles and runs**

The demo uses `widgets.NewImage(generateDemoGradient(), "gradient")`. Since `NewImage` still accepts raw bytes, the demo should compile without changes. Verify:

Run: `go build ./apps/texelui-demo/...`
Expected: compiles

If there are any compilation errors from removed types (like `ImagePlacement`), fix them. The `generateDemoGradient()` function returns PNG bytes which `NewImage` decodes — this should still work.

**Step 2: Build demo binary**

Run: `make demos`
Expected: `bin/texelui-demo` built

**Step 3: Commit (if changes needed)**

```bash
git add apps/texelui-demo/demo.go
git commit -m "Update demo for GraphicsProvider v2"
```

---

### Task 7: Clean up old test files and verify full test suite

**Step 1: Remove old tests for removed API**

Remove tests that reference `ImagePlacement`, old `PlaceImage`, old `DeleteImage`, old `DeleteAll` from `core/graphics_test.go`. Replace with tests for the new API (done in Task 1).

**Step 2: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

**Step 3: Run with race detector**

Run: `go test -race ./...`
Expected: ALL PASS, no races

**Step 4: Commit**

```bash
git add -A
git commit -m "Clean up old GraphicsProvider v1 tests"
```

---

## Phase 2: Texelation Protocol + Server

### Task 8: Add protocol message types

**Files:**
- Modify: `texelation/protocol/protocol.go` — add message type constants
- Create: `texelation/protocol/image_messages.go` — encode/decode functions
- Create: `texelation/protocol/image_messages_test.go` — round-trip tests

**Step 1: Add message type constants**

In `texelation/protocol/protocol.go`, add after `MsgClientReady`:

```go
MsgImageUpload
MsgImagePlace
MsgImageDelete
MsgImageReset
```

These will be iota values 25-28.

**Step 2: Write failing tests**

```go
package protocol

import (
	"testing"
)

func TestImageUploadRoundTrip(t *testing.T) {
	original := ImageUpload{
		PaneID:    [16]byte{1, 2, 3},
		SurfaceID: 42,
		Width:     320,
		Height:    240,
		Format:    0,
		Data:      []byte("fake-png-data-here"),
	}
	encoded, err := EncodeImageUpload(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeImageUpload(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.PaneID != original.PaneID {
		t.Errorf("PaneID mismatch")
	}
	if decoded.SurfaceID != original.SurfaceID {
		t.Errorf("SurfaceID: got %d, want %d", decoded.SurfaceID, original.SurfaceID)
	}
	if decoded.Width != original.Width || decoded.Height != original.Height {
		t.Errorf("dimensions: got %dx%d, want %dx%d", decoded.Width, decoded.Height, original.Width, original.Height)
	}
	if string(decoded.Data) != string(original.Data) {
		t.Errorf("data mismatch")
	}
}

func TestImagePlaceRoundTrip(t *testing.T) {
	original := ImagePlace{
		PaneID:    [16]byte{4, 5, 6},
		SurfaceID: 7,
		X:         10,
		Y:         20,
		W:         30,
		H:         15,
		ZIndex:    -1,
	}
	encoded, err := EncodeImagePlace(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeImagePlace(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != original {
		t.Errorf("got %+v, want %+v", decoded, original)
	}
}

func TestImageDeleteRoundTrip(t *testing.T) {
	original := ImageDelete{
		PaneID:    [16]byte{7, 8, 9},
		SurfaceID: 99,
	}
	encoded, err := EncodeImageDelete(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeImageDelete(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != original {
		t.Errorf("got %+v, want %+v", decoded, original)
	}
}

func TestImageResetRoundTrip(t *testing.T) {
	original := ImageReset{
		PaneID: [16]byte{10, 11, 12},
	}
	encoded, err := EncodeImageReset(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeImageReset(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != original {
		t.Errorf("got %+v, want %+v", decoded, original)
	}
}
```

**Step 3: Implement encode/decode**

Create `texelation/protocol/image_messages.go`:

```go
package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// ImageUpload carries image data from server to client.
type ImageUpload struct {
	PaneID    [16]byte
	SurfaceID uint32
	Width     uint16
	Height    uint16
	Format    uint8 // 0=PNG
	Data      []byte
}

// ImagePlace tells the client to display a cached image.
type ImagePlace struct {
	PaneID    [16]byte
	SurfaceID uint32
	X, Y      uint16
	W, H      uint16
	ZIndex    int8
}

// ImageDelete tells the client to free image data.
type ImageDelete struct {
	PaneID    [16]byte
	SurfaceID uint32
}

// ImageReset tells the client to clear all placements for a pane.
type ImageReset struct {
	PaneID [16]byte
}

func EncodeImageUpload(u ImageUpload) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(u.PaneID[:])
	binary.Write(&buf, binary.LittleEndian, u.SurfaceID)
	binary.Write(&buf, binary.LittleEndian, u.Width)
	binary.Write(&buf, binary.LittleEndian, u.Height)
	buf.WriteByte(u.Format)
	dataLen := uint32(len(u.Data))
	binary.Write(&buf, binary.LittleEndian, dataLen)
	buf.Write(u.Data)
	return buf.Bytes(), nil
}

func DecodeImageUpload(b []byte) (ImageUpload, error) {
	if len(b) < 29 { // 16+4+2+2+1+4
		return ImageUpload{}, fmt.Errorf("image upload too short: %d", len(b))
	}
	var u ImageUpload
	copy(u.PaneID[:], b[:16])
	u.SurfaceID = binary.LittleEndian.Uint32(b[16:20])
	u.Width = binary.LittleEndian.Uint16(b[20:22])
	u.Height = binary.LittleEndian.Uint16(b[22:24])
	u.Format = b[24]
	dataLen := binary.LittleEndian.Uint32(b[25:29])
	if len(b) < 29+int(dataLen) {
		return ImageUpload{}, fmt.Errorf("image upload data truncated")
	}
	u.Data = make([]byte, dataLen)
	copy(u.Data, b[29:29+dataLen])
	return u, nil
}

func EncodeImagePlace(p ImagePlace) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(p.PaneID[:])
	binary.Write(&buf, binary.LittleEndian, p.SurfaceID)
	binary.Write(&buf, binary.LittleEndian, p.X)
	binary.Write(&buf, binary.LittleEndian, p.Y)
	binary.Write(&buf, binary.LittleEndian, p.W)
	binary.Write(&buf, binary.LittleEndian, p.H)
	binary.Write(&buf, binary.LittleEndian, p.ZIndex)
	return buf.Bytes(), nil
}

func DecodeImagePlace(b []byte) (ImagePlace, error) {
	if len(b) < 29 { // 16+4+2+2+2+2+1
		return ImagePlace{}, fmt.Errorf("image place too short: %d", len(b))
	}
	var p ImagePlace
	copy(p.PaneID[:], b[:16])
	p.SurfaceID = binary.LittleEndian.Uint32(b[16:20])
	p.X = binary.LittleEndian.Uint16(b[20:22])
	p.Y = binary.LittleEndian.Uint16(b[22:24])
	p.W = binary.LittleEndian.Uint16(b[24:26])
	p.H = binary.LittleEndian.Uint16(b[26:28])
	p.ZIndex = int8(b[28])
	return p, nil
}

func EncodeImageDelete(d ImageDelete) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(d.PaneID[:])
	binary.Write(&buf, binary.LittleEndian, d.SurfaceID)
	return buf.Bytes(), nil
}

func DecodeImageDelete(b []byte) (ImageDelete, error) {
	if len(b) < 20 {
		return ImageDelete{}, fmt.Errorf("image delete too short: %d", len(b))
	}
	var d ImageDelete
	copy(d.PaneID[:], b[:16])
	d.SurfaceID = binary.LittleEndian.Uint32(b[16:20])
	return d, nil
}

func EncodeImageReset(r ImageReset) ([]byte, error) {
	return r.PaneID[:], nil
}

func DecodeImageReset(b []byte) (ImageReset, error) {
	if len(b) < 16 {
		return ImageReset{}, fmt.Errorf("image reset too short: %d", len(b))
	}
	var r ImageReset
	copy(r.PaneID[:], b[:16])
	return r, nil
}
```

**Step 4: Run tests**

Run: `go test ./protocol/ -run TestImage -v`
Expected: PASS

**Step 5: Commit**

```bash
git add protocol/protocol.go protocol/image_messages.go protocol/image_messages_test.go
git commit -m "Add image protocol messages: Upload, Place, Delete, Reset"
```

---

### Task 9: Implement RemoteGraphicsProvider

**Files:**
- Create: `texelation/internal/runtime/server/remote_graphics.go`
- Create: `texelation/internal/runtime/server/remote_graphics_test.go`

**Step 1: Write failing tests**

```go
package server

import (
	"image"
	"image/color"
	"sync"
	"testing"

	"github.com/framegrace/texelation/protocol"
	"github.com/framegrace/texelui/core"
)

type capturedMessage struct {
	msgType uint8
	payload []byte
}

func TestRemoteGraphicsCreateSurface(t *testing.T) {
	var msgs []capturedMessage
	var mu sync.Mutex
	sender := func(msgType uint8, payload []byte) {
		mu.Lock()
		msgs = append(msgs, capturedMessage{msgType, payload})
		mu.Unlock()
	}

	gp := NewRemoteGraphicsProvider([16]byte{1}, sender)
	s := gp.CreateSurface(100, 50)
	if s == nil {
		t.Fatal("CreateSurface returned nil")
	}
	if s.ID() == 0 {
		t.Error("expected non-zero ID")
	}
	if s.Buffer().Bounds().Dx() != 100 {
		t.Errorf("width: got %d, want 100", s.Buffer().Bounds().Dx())
	}
}

func TestRemoteGraphicsUpdateSendsUpload(t *testing.T) {
	var msgs []capturedMessage
	var mu sync.Mutex
	sender := func(msgType uint8, payload []byte) {
		mu.Lock()
		msgs = append(msgs, capturedMessage{msgType, payload})
		mu.Unlock()
	}

	gp := NewRemoteGraphicsProvider([16]byte{1}, sender)
	s := gp.CreateSurface(4, 2)
	s.Buffer().Set(0, 0, color.RGBA{255, 0, 0, 255})

	if err := s.Update(); err != nil {
		t.Fatalf("Update: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].msgType != protocol.MsgImageUpload {
		t.Errorf("expected MsgImageUpload, got %d", msgs[0].msgType)
	}
	// Verify we can decode the payload
	upload, err := protocol.DecodeImageUpload(msgs[0].payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if upload.SurfaceID != s.ID() {
		t.Errorf("surface ID: got %d, want %d", upload.SurfaceID, s.ID())
	}
	if upload.Width != 4 || upload.Height != 2 {
		t.Errorf("dimensions: got %dx%d, want 4x2", upload.Width, upload.Height)
	}
	if len(upload.Data) == 0 {
		t.Error("expected non-empty PNG data")
	}
}

func TestRemoteGraphicsPlaceSendsPlace(t *testing.T) {
	var msgs []capturedMessage
	var mu sync.Mutex
	sender := func(msgType uint8, payload []byte) {
		mu.Lock()
		msgs = append(msgs, capturedMessage{msgType, payload})
		mu.Unlock()
	}

	gp := NewRemoteGraphicsProvider([16]byte{2}, sender)
	s := gp.CreateSurface(10, 10)

	painter := core.NewPainter(make([][]core.Cell, 24), core.Rect{X: 0, Y: 0, W: 80, H: 24})
	s.Place(painter, core.Rect{X: 5, Y: 10, W: 20, H: 8}, -1)

	mu.Lock()
	defer mu.Unlock()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].msgType != protocol.MsgImagePlace {
		t.Errorf("expected MsgImagePlace, got %d", msgs[0].msgType)
	}
	place, err := protocol.DecodeImagePlace(msgs[0].payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if place.X != 5 || place.Y != 10 || place.W != 20 || place.H != 8 {
		t.Errorf("rect mismatch: %+v", place)
	}
	if place.ZIndex != -1 {
		t.Errorf("zIndex: got %d, want -1", place.ZIndex)
	}
}

func TestRemoteGraphicsResetSendsReset(t *testing.T) {
	var msgs []capturedMessage
	var mu sync.Mutex
	sender := func(msgType uint8, payload []byte) {
		mu.Lock()
		msgs = append(msgs, capturedMessage{msgType, payload})
		mu.Unlock()
	}

	paneID := [16]byte{3}
	gp := NewRemoteGraphicsProvider(paneID, sender)
	gp.Reset()

	mu.Lock()
	defer mu.Unlock()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].msgType != protocol.MsgImageReset {
		t.Errorf("expected MsgImageReset, got %d", msgs[0].msgType)
	}
}

func TestRemoteGraphicsCapability(t *testing.T) {
	gp := NewRemoteGraphicsProvider([16]byte{}, nil)
	// Remote provider reports Kitty since the client will handle rendering
	if gp.Capability() != core.GraphicsKitty {
		t.Errorf("got %d, want GraphicsKitty", gp.Capability())
	}
}
```

**Step 2: Implement RemoteGraphicsProvider**

```go
package server

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"sync"

	"github.com/framegrace/texelation/protocol"
	"github.com/framegrace/texelui/core"
)

// MessageSender sends an encoded protocol message to the client.
type MessageSender func(msgType uint8, payload []byte)

// remoteSurface implements core.ImageSurface for the remote provider.
type remoteSurface struct {
	provider *RemoteGraphicsProvider
	id       uint32
	buf      *image.RGBA
	width    uint16
	height   uint16
}

func (s *remoteSurface) ID() uint32          { return s.id }
func (s *remoteSurface) Buffer() *image.RGBA { return s.buf }

func (s *remoteSurface) Update() error {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, s.buf); err != nil {
		return fmt.Errorf("png encode: %w", err)
	}
	payload, err := protocol.EncodeImageUpload(protocol.ImageUpload{
		PaneID:    s.provider.paneID,
		SurfaceID: s.id,
		Width:     s.width,
		Height:    s.height,
		Format:    0, // PNG
		Data:      pngBuf.Bytes(),
	})
	if err != nil {
		return fmt.Errorf("encode image upload: %w", err)
	}
	s.provider.send(protocol.MsgImageUpload, payload)
	return nil
}

func (s *remoteSurface) Place(p *core.Painter, rect core.Rect, zIndex int) {
	payload, _ := protocol.EncodeImagePlace(protocol.ImagePlace{
		PaneID:    s.provider.paneID,
		SurfaceID: s.id,
		X:         uint16(rect.X),
		Y:         uint16(rect.Y),
		W:         uint16(rect.W),
		H:         uint16(rect.H),
		ZIndex:    int8(zIndex),
	})
	s.provider.send(protocol.MsgImagePlace, payload)
}

func (s *remoteSurface) Delete() {
	payload, _ := protocol.EncodeImageDelete(protocol.ImageDelete{
		PaneID:    s.provider.paneID,
		SurfaceID: s.id,
	})
	s.provider.send(protocol.MsgImageDelete, payload)
	s.buf = nil
}

// RemoteGraphicsProvider sends graphics commands to a texelation client.
type RemoteGraphicsProvider struct {
	mu     sync.Mutex
	paneID [16]byte
	send   MessageSender
	nextID uint32
}

// NewRemoteGraphicsProvider creates a provider that sends messages via sender.
func NewRemoteGraphicsProvider(paneID [16]byte, sender MessageSender) *RemoteGraphicsProvider {
	return &RemoteGraphicsProvider{
		paneID: paneID,
		send:   sender,
		nextID: 1,
	}
}

func (p *RemoteGraphicsProvider) Capability() core.GraphicsCapability {
	return core.GraphicsKitty
}

func (p *RemoteGraphicsProvider) CreateSurface(width, height int) core.ImageSurface {
	p.mu.Lock()
	defer p.mu.Unlock()
	id := p.nextID
	p.nextID++
	return &remoteSurface{
		provider: p,
		id:       id,
		buf:      image.NewRGBA(image.Rect(0, 0, width, height)),
		width:    uint16(width),
		height:   uint16(height),
	}
}

func (p *RemoteGraphicsProvider) Reset() {
	payload, _ := protocol.EncodeImageReset(protocol.ImageReset{
		PaneID: p.paneID,
	})
	p.send(protocol.MsgImageReset, payload)
}
```

**Step 3: Run tests**

Run: `go test ./internal/runtime/server/ -run TestRemoteGraphics -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/runtime/server/remote_graphics.go internal/runtime/server/remote_graphics_test.go
git commit -m "Add RemoteGraphicsProvider for texelation server"
```

---

### Task 10: Wire RemoteGraphicsProvider into pane creation

**Files:**
- Modify: `texelation/texel/pane.go` — inject provider in `AttachApp()`
- Modify: `texelation/internal/runtime/server/session.go` — add `EnqueueImage` method

**Step 1: Add EnqueueImage to Session**

Follow the `EnqueueDiff` pattern. Add a method that enqueues an image message:

```go
func (s *Session) EnqueueImage(msgType uint8, payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	seq := s.nextSequence + 1
	s.nextSequence = seq
	hdr := protocol.Header{
		Version:   protocol.Version,
		Type:      msgType,
		SessionID: s.id,
		Sequence:  seq,
	}
	s.diffs = append(s.diffs, DiffPacket{
		Sequence: seq,
		Payload:  payload,
		Message:  hdr,
	})
	if s.maxDiffs > 0 && len(s.diffs) > s.maxDiffs {
		drop := len(s.diffs) - s.maxDiffs
		s.recordDrop(drop)
		s.diffs = append([]DiffPacket(nil), s.diffs[drop:]...)
	}
	return nil
}
```

**Step 2: Inject RemoteGraphicsProvider in AttachApp**

In `texel/pane.go`, in the `AttachApp` method, after the clipboard injection (~line 169), add:

```go
// Provide graphics service
if gs, ok := app.(interface{ SetGraphicsProvider(core.GraphicsProvider) }); ok {
	sender := func(msgType uint8, payload []byte) {
		// Route through pane's session
		if sess := p.screen.desktop.ActiveSession(); sess != nil {
			sess.EnqueueImage(msgType, payload)
		}
	}
	gs.SetGraphicsProvider(server.NewRemoteGraphicsProvider(p.id, sender))
}
```

Note: The exact wiring depends on how the pane accesses the session. Check `p.screen.desktop` for the session accessor. The sender callback needs to route messages to all connected clients. Look at how `DesktopPublisher.Publish()` uses `session.EnqueueDiff()` and follow the same pattern.

If the app implements `interface{ UI() *core.UIManager }` (like `adapter.UIApp` does), inject via:

```go
if ua, ok := app.(interface{ UI() *core.UIManager }); ok {
	sender := func(msgType uint8, payload []byte) { /* ... */ }
	ua.UI().SetGraphicsProvider(server.NewRemoteGraphicsProvider(p.id, sender))
}
```

**Step 3: Build and test**

Run: `go build ./...`
Expected: compiles

**Step 4: Commit**

```bash
git add texel/pane.go internal/runtime/server/session.go
git commit -m "Wire RemoteGraphicsProvider into pane AttachApp"
```

---

## Phase 3: Texelation Client

### Task 11: Implement client-side ImageCache

**Files:**
- Create: `texelation/client/image_cache.go`
- Create: `texelation/client/image_cache_test.go`

**Step 1: Write failing tests**

```go
package client

import (
	"testing"
)

func TestImageCacheUpload(t *testing.T) {
	cache := NewImageCache()
	paneID := [16]byte{1}
	cache.Upload(paneID, 42, 100, 50, []byte("fake-png"))

	img := cache.Get(42)
	if img == nil {
		t.Fatal("expected cached image")
	}
	if string(img.Data) != "fake-png" {
		t.Error("data mismatch")
	}
}

func TestImageCachePlaceAndPlacements(t *testing.T) {
	cache := NewImageCache()
	paneID := [16]byte{1}
	cache.Upload(paneID, 1, 10, 10, []byte("png"))

	cache.Place(paneID, 1, 5, 10, 20, 8, -1)

	placements := cache.Placements(paneID)
	if len(placements) != 1 {
		t.Fatalf("expected 1 placement, got %d", len(placements))
	}
	if placements[0].SurfaceID != 1 {
		t.Errorf("surface ID: got %d, want 1", placements[0].SurfaceID)
	}
}

func TestImageCacheReset(t *testing.T) {
	cache := NewImageCache()
	paneID := [16]byte{1}
	cache.Upload(paneID, 1, 10, 10, []byte("png"))
	cache.Place(paneID, 1, 0, 0, 10, 10, 0)
	cache.ResetPlacements(paneID)

	placements := cache.Placements(paneID)
	if len(placements) != 0 {
		t.Errorf("expected 0 placements after reset, got %d", len(placements))
	}
	// Image data should still be cached
	if cache.Get(1) == nil {
		t.Error("image data should be preserved after reset")
	}
}

func TestImageCacheDelete(t *testing.T) {
	cache := NewImageCache()
	paneID := [16]byte{1}
	cache.Upload(paneID, 1, 10, 10, []byte("png"))
	cache.Place(paneID, 1, 0, 0, 10, 10, 0)
	cache.Delete(paneID, 1)

	if cache.Get(1) != nil {
		t.Error("expected nil after delete")
	}
	placements := cache.Placements(paneID)
	if len(placements) != 0 {
		t.Error("expected placements cleared after delete")
	}
}
```

**Step 2: Implement ImageCache**

```go
package client

import (
	"bytes"
	"image"
	_ "image/png"
	"sync"
)

// CachedImage holds uploaded image data and a decoded form for half-block fallback.
type CachedImage struct {
	PaneID  [16]byte
	Data    []byte      // PNG bytes from server
	Decoded image.Image // decoded once for half-block
	Width   int
	Height  int
}

// ImagePlacement describes where to display a cached image.
type ImagePlacement struct {
	SurfaceID uint32
	X, Y      int
	W, H      int
	ZIndex    int
}

// ImageCache stores uploaded images and active placements per pane.
type ImageCache struct {
	mu         sync.RWMutex
	images     map[uint32]*CachedImage
	placements map[[16]byte][]ImagePlacement
}

func NewImageCache() *ImageCache {
	return &ImageCache{
		images:     make(map[uint32]*CachedImage),
		placements: make(map[[16]byte][]ImagePlacement),
	}
}

func (c *ImageCache) Upload(paneID [16]byte, surfaceID uint32, width, height int, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	decoded, _, _ := image.Decode(bytes.NewReader(data))
	c.images[surfaceID] = &CachedImage{
		PaneID:  paneID,
		Data:    data,
		Decoded: decoded,
		Width:   width,
		Height:  height,
	}
}

func (c *ImageCache) Get(surfaceID uint32) *CachedImage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.images[surfaceID]
}

func (c *ImageCache) Place(paneID [16]byte, surfaceID uint32, x, y, w, h, zIndex int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.placements[paneID] = append(c.placements[paneID], ImagePlacement{
		SurfaceID: surfaceID,
		X: x, Y: y, W: w, H: h,
		ZIndex: zIndex,
	})
}

func (c *ImageCache) Placements(paneID [16]byte) []ImagePlacement {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.placements[paneID]
}

func (c *ImageCache) ResetPlacements(paneID [16]byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.placements[paneID] = nil
}

func (c *ImageCache) Delete(paneID [16]byte, surfaceID uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.images, surfaceID)
	// Remove placements for this surface
	pls := c.placements[paneID]
	filtered := pls[:0]
	for _, p := range pls {
		if p.SurfaceID != surfaceID {
			filtered = append(filtered, p)
		}
	}
	c.placements[paneID] = filtered
}
```

**Step 3: Run tests**

Run: `go test ./client/ -run TestImageCache -v`
Expected: PASS

**Step 4: Commit**

```bash
git add client/image_cache.go client/image_cache_test.go
git commit -m "Add client-side ImageCache for texelation"
```

---

### Task 12: Add client message handling

**Files:**
- Modify: `texelation/internal/runtime/client/protocol_handler.go`

**Step 1: Add image message cases**

In the `handleControlMessage` switch statement, add cases for the four image message types:

```go
case protocol.MsgImageUpload:
	upload, err := protocol.DecodeImageUpload(payload)
	if err != nil {
		return false
	}
	cache.ImageCache().Upload(upload.PaneID, upload.SurfaceID,
		int(upload.Width), int(upload.Height), upload.Data)
	return true

case protocol.MsgImagePlace:
	place, err := protocol.DecodeImagePlace(payload)
	if err != nil {
		return false
	}
	cache.ImageCache().Place(place.PaneID, place.SurfaceID,
		int(place.X), int(place.Y), int(place.W), int(place.H), int(place.ZIndex))
	return true

case protocol.MsgImageDelete:
	del, err := protocol.DecodeImageDelete(payload)
	if err != nil {
		return false
	}
	cache.ImageCache().Delete(del.PaneID, del.SurfaceID)
	return true

case protocol.MsgImageReset:
	reset, err := protocol.DecodeImageReset(payload)
	if err != nil {
		return false
	}
	cache.ImageCache().ResetPlacements(reset.PaneID)
	return true
```

**Note**: The `cache` variable in the handler needs access to an `ImageCache()` method. If `BufferCache` doesn't have one, add it:

```go
func (c *BufferCache) ImageCache() *ImageCache { return c.imageCache }
```

And initialize `imageCache` in `NewBufferCache()`.

**Step 2: Build**

Run: `go build ./...`
Expected: compiles

**Step 3: Commit**

```bash
git add internal/runtime/client/protocol_handler.go client/buffercache.go
git commit -m "Handle image protocol messages in client"
```

---

### Task 13: Integrate images into client renderer

**Files:**
- Modify: `texelation/internal/runtime/client/renderer.go`

**Step 1: Add half-block rendering for non-Kitty clients**

After the pane cell compositing loop and before `showWorkspaceBuffer`, add image rendering:

```go
// Render images for each pane
for _, pane := range panes {
	placements := state.cache.ImageCache().Placements(pane.ID)
	for _, pl := range placements {
		img := state.cache.ImageCache().Get(pl.SurfaceID)
		if img == nil || img.Decoded == nil {
			continue
		}
		// Render half-block art into workspace buffer at pane position + image offset
		renderHalfBlockIntoBuffer(workspaceBuffer, img.Decoded,
			pane.Rect.X+pl.X, pane.Rect.Y+pl.Y, pl.W, pl.H)
	}
}
```

Implement `renderHalfBlockIntoBuffer` using the same half-block algorithm as `HalfBlockProvider.Place`:

```go
func renderHalfBlockIntoBuffer(buf [][]Cell, img image.Image, screenX, screenY, w, h int) {
	imgBounds := img.Bounds()
	imgW := imgBounds.Dx()
	imgH := imgBounds.Dy()
	pixW := w
	pixH := h * 2

	for cy := range h {
		row := screenY + cy
		if row < 0 || row >= len(buf) {
			continue
		}
		for cx := range w {
			col := screenX + cx
			if col < 0 || col >= len(buf[row]) {
				continue
			}
			topPixY := cy * 2
			botPixY := cy*2 + 1

			topR, topG, topB := sampleColor(img, cx, topPixY, pixW, pixH, imgW, imgH)
			botR, botG, botB := sampleColor(img, cx, botPixY, pixW, pixH, imgW, imgH)

			style := tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(int32(topR), int32(topG), int32(topB))).
				Background(tcell.NewRGBColor(int32(botR), int32(botG), int32(botB)))

			buf[row][col] = Cell{Ch: '\u2580', Style: style}
		}
	}
}

func sampleColor(img image.Image, cx, py, pixW, pixH, imgW, imgH int) (uint8, uint8, uint8) {
	imgX := cx * imgW / pixW
	imgY := py * imgH / pixH
	if imgX >= imgW { imgX = imgW - 1 }
	if imgY >= imgH { imgY = imgH - 1 }
	bounds := img.Bounds()
	r, g, b, _ := img.At(bounds.Min.X+imgX, bounds.Min.Y+imgY).RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}
```

**Step 2: Add Kitty rendering for Kitty-capable clients**

After `screen.Show()`, add Kitty flush:

```go
// Flush Kitty image placements (if terminal supports it)
if state.kittyProvider != nil {
	for _, pane := range panes {
		placements := state.cache.ImageCache().Placements(pane.ID)
		for _, pl := range placements {
			img := state.cache.ImageCache().Get(pl.SurfaceID)
			if img == nil {
				continue
			}
			state.kittyProvider.PlaceFromCache(pl.SurfaceID,
				pane.Rect.X+pl.X, pane.Rect.Y+pl.Y, pl.W, pl.H, pl.ZIndex)
		}
	}
	if tty, ok := screen.Tty(); ok {
		state.kittyProvider.Flush(tty)
	}
}
```

Note: The client needs a local `KittyProvider` that can transmit + place. On `ImageUpload`, the client should also transmit to the terminal. Add a `TransmitCached(id uint32, pngData []byte)` method to the client's Kitty provider, or reuse the surface `Update()` flow.

This is the most complex integration point. The key insight: the client doesn't use `ImageSurface` — it uses the `ImageCache` for storage and writes Kitty APC commands directly using the cached PNG data. The client-side Kitty logic is a thin wrapper that:
1. On `ImageUpload` → calls `writeTransmit(tty, surfaceID, pngData)` from `graphics/kitty.go`
2. On `ImagePlace` → calls `writePut(tty, surfaceID, rect, zIndex)` from `graphics/kitty.go`
3. On `ImageReset` → calls `writeResetPlacements(tty)` from `graphics/kitty.go`
4. On `ImageDelete` → calls `writeDeleteByID(tty, surfaceID)` from `graphics/kitty.go`

To reuse these functions, export them from `graphics/kitty.go` (capitalize: `WriteTransmit`, `WritePut`, `WriteResetPlacements`, `WriteDeleteByID`).

**Step 3: Add Kitty detection at client startup**

In the client startup code, detect capability and create a local provider:

```go
if graphics.DetectCapability() == core.GraphicsKitty {
	state.kittyProvider = graphics.NewKittyProvider()
}
```

**Step 4: Build and manual test**

Run: `go build ./...`
Expected: compiles

**Step 5: Commit**

```bash
git add internal/runtime/client/renderer.go
git commit -m "Integrate image rendering into texelation client renderer"
```

---

### Task 14: Register texelui-demo in texelation

**Files:**
- Create: `texelation/apps/texeluidemo/register.go`
- Modify: `texelation/cmd/texel-server/main.go` — add import

**Step 1: Create registration file**

```go
package texeluidemo

import (
	texeluidemo "github.com/framegrace/texelui/apps/texelui-demo"
	"github.com/framegrace/texelation/registry"
)

func init() {
	registry.RegisterBuiltInProvider(func(_ *registry.Registry) (*registry.Manifest, registry.AppFactory) {
		return &registry.Manifest{
			Name:        "texelui-demo",
			DisplayName: "TexelUI Demo",
			Description: "Widget showcase with graphics demo",
			Icon:        "🧩",
			Category:    "demo",
		}, func() interface{} {
			return texeluidemo.New()
		}
	})
}
```

**Step 2: Add import to texel-server main.go**

Add to the import block:
```go
_ "github.com/framegrace/texelation/apps/texeluidemo"
```

**Step 3: Build and test**

Run: `go build ./cmd/texel-server/...`
Expected: compiles

Test: Launch texel-server, open launcher, verify "TexelUI Demo" appears. Launch it. Navigate to Widgets tab. Verify image renders (half-block on non-Kitty, Kitty graphics on Kitty terminal).

**Step 4: Commit**

```bash
git add apps/texeluidemo/register.go cmd/texel-server/main.go
git commit -m "Register texelui-demo in texelation app registry"
```

---

## Phase 4: Verification

### Task 15: End-to-end verification

**Step 1: Standalone mode**

```bash
cd texelui
make demos
./bin/texelui-demo
```

Verify:
- [ ] Navigate to Widgets tab
- [ ] Image renders (half-block or Kitty depending on terminal)
- [ ] Switching tabs hides the image
- [ ] Switching back shows it without re-upload flicker
- [ ] Label shows "Image (kitty):" or "Image (block art):" correctly

**Step 2: Texelation mode**

```bash
cd texelation
make build
./bin/texel-server
```

In the launcher, open "TexelUI Demo". Verify same behavior as standalone:
- [ ] Image renders in the Widgets tab
- [ ] Tab switching hides/shows correctly
- [ ] No rendering artifacts when pane is moved or resized
- [ ] Works on both Kitty and non-Kitty client terminals

**Step 3: Run full test suites**

```bash
cd texelui && go test -race ./...
cd texelation && go test -race ./...
```

Expected: ALL PASS, no races

**Step 4: Commit any fixes**

```bash
git add -A
git commit -m "Fix issues found during end-to-end verification"
```
