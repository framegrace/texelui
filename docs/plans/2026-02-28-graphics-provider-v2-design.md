# GraphicsProvider v2: Upload/Place Separation

**Date:** 2026-02-28
**Status:** Approved
**Scope:** TexelUI core API, standalone runtime, texelation server + client

## Problem

The current `GraphicsProvider` API conflates image upload and display into a single `PlaceImage(ImagePlacement)` call that sends full image bytes every frame. This works for standalone apps but is unsuitable for texelation's client/server model, where sending image data over the wire every frame wastes bandwidth. It also prevents future shared-memory optimization (zero-copy rendering).

## Decision

Redesign the API around an `ImageSurface` object that separates image upload from placement. The app renders into a surface buffer, calls `Update()` when content changes (upload), and calls `Place()` every frame (lightweight). Three provider implementations serve all modes transparently: `KittyProvider` (standalone Kitty), `HalfBlockProvider` (standalone fallback), and `RemoteGraphicsProvider` (texelation server). The client detects its own terminal capability and renders images accordingly.

## Core API

```go
// core/graphics.go

type GraphicsCapability int

const (
    GraphicsNone      GraphicsCapability = iota
    GraphicsHalfBlock
    GraphicsKitty
)

type GraphicsProvider interface {
    Capability() GraphicsCapability
    CreateSurface(width, height int) ImageSurface
    Reset() // clear all active placements (cached data preserved)
}

type ImageSurface interface {
    ID() uint32
    Buffer() *image.RGBA                      // direct pixel access
    Update() error                            // signal content changed
    Place(p *Painter, rect Rect, zIndex int)  // queue display at position
    Delete()                                  // free data + placements
}
```

### Semantics

| Method | What it does | When called | Cost |
|--------|-------------|-------------|------|
| `CreateSurface(w, h)` | Allocate pixel buffer, assign ID | Widget init / resize | Once |
| `Buffer()` | Return `*image.RGBA` for direct rendering | Before Update | Free |
| `Update()` | Encode + transmit image data | When content changes | Heavy |
| `Place(p, rect, z)` | Queue display at position | Every frame during Draw | Cheap |
| `Reset()` | Clear all active placements | Before each render | Cheap |
| `Delete()` | Free image data + placements | Widget destroyed | Once |

### Kitty Protocol Mapping

| API call | Kitty command |
|----------|--------------|
| `Update()` | `a=t` (transmit, no display) |
| `Place()` | `a=p,i=<id>,c=<cols>,r=<rows>` (put from cache) |
| `Reset()` | `a=d,d=a,q=2` (delete placements, keep data) |
| `Delete()` | `a=d,d=I,i=<id>,q=2` (delete data + placements) |

## App Developer Experience

The app developer only uses `core.GraphicsProvider` and `core.ImageSurface`, obtained from the Painter during `Draw()`. Same code runs unchanged in standalone and texelation:

```go
type Minimap struct {
    core.BaseWidget
    surface core.ImageSurface
    dirty   bool
}

func (m *Minimap) Draw(p *core.Painter) {
    gp := p.GraphicsProvider()
    if gp == nil {
        return
    }

    if m.surface == nil {
        m.surface = gp.CreateSurface(m.Rect.W*8, m.Rect.H*16)
    }

    if m.dirty {
        renderMinimap(m.surface.Buffer()) // any library: Cairo, gg, stdlib
        m.surface.Update()
        m.dirty = false
    }

    m.surface.Place(p, m.Rect, -1)
}
```

The app never imports `graphics/`, `runtime/`, or `protocol/`. The runtime or texelation injects the right provider. The developer's contract:

1. Get provider from painter
2. Create surface, render into buffer with any 2D library
3. Call `Update()` when content changes
4. Call `Place()` every frame

## Provider Implementations

### KittyProvider (standalone, Kitty-capable terminal)

- `CreateSurface(w, h)` — allocates `image.RGBA`, assigns unique ID
- `Buffer()` — returns the RGBA buffer directly
- `Update()` — encodes to PNG, queues `a=t` (transmit only) APC command
- `Place(p, rect, z)` — fills rect with spaces (clear area for tcell), queues `a=p` APC command
- `Reset()` — queues `a=d,d=a` (delete placements, keep data)
- `Delete()` — queues `a=d,d=I,i=<id>` (delete data + placements)
- `Flush(tty)` — writes all queued commands to TTY after `screen.Show()`

### HalfBlockProvider (standalone, no Kitty)

- `CreateSurface(w, h)` — allocates `image.RGBA`, assigns ID
- `Buffer()` — returns the RGBA buffer
- `Update()` — no-op (data lives in the buffer, read on Place)
- `Place(p, rect, z)` — reads buffer, renders half-block characters via painter
- `Reset()` — no-op (cell-based, tcell clears on screen.Clear)
- `Delete()` — frees buffer

### RemoteGraphicsProvider (texelation server-side)

- `CreateSurface(w, h)` — allocates `image.RGBA`, assigns ID
- `Buffer()` — returns the RGBA buffer
- `Update()` — encodes to PNG, sends `MsgImageUpload` to client
- `Place(p, rect, z)` — sends `MsgImagePlace` to client (ignores painter)
- `Reset()` — sends `MsgImageReset` to client
- `Delete()` — sends `MsgImageDelete` to client

## Texelation Protocol Messages

Four new message types following the existing binary format:

```go
// MsgImageUpload — server sends image data to client (on surface.Update())
type ImageUpload struct {
    PaneID    [16]byte
    SurfaceID uint32
    Width     uint16
    Height    uint16
    Format    uint8   // 0=PNG
    Data      []byte  // encoded image bytes
}

// MsgImagePlace — display a cached image (on surface.Place())
type ImagePlace struct {
    PaneID    [16]byte
    SurfaceID uint32
    X, Y      uint16  // relative to pane origin
    W, H      uint16  // display size in cells
    ZIndex    int8
}

// MsgImageDelete — free a specific image (on surface.Delete())
type ImageDelete struct {
    PaneID    [16]byte
    SurfaceID uint32
}

// MsgImageReset — clear all placements for a pane (on provider.Reset())
type ImageReset struct {
    PaneID [16]byte
}
```

`ImageUpload` is the heavy message (carries image bytes) — only sent when content changes. `ImagePlace` is ~20 bytes — sent every frame for visible images.

## Client-Side Rendering

The texelation client detects its own terminal capability at startup using `graphics.DetectCapability()` and renders images accordingly.

### Image Cache

```go
type ImageCache struct {
    images     map[uint32]*cachedImage         // by surface ID
    placements map[[16]byte][]ImagePlacement   // by pane ID
}

type cachedImage struct {
    paneID  [16]byte
    data    []byte       // PNG bytes from server
    decoded image.Image  // decoded once, for half-block fallback
}
```

### Message Handling

| Message | Handler |
|---------|---------|
| `ImageUpload` | Cache PNG + decode `image.Image` for fallback |
| `ImagePlace` | Add to pane's placement list |
| `ImageReset` | Clear pane's placement list |
| `ImageDelete` | Remove from cache and placements |

### Render Integration

Current client render flow:

```
Clear → composite pane cells → write to tcell → Show()
```

New flow:

```
Clear → composite pane cells → render half-block images* → write to tcell → Show() → flush Kitty placements**
```

`*` Non-Kitty clients only: render half-block art into workspace cell buffer at pane offset + image rect.

`**` Kitty clients only: queue `a=p` APC commands for each placement using cached image data, flush after Show().

## Integration Points

### Standalone Runtime (`runtime/runner.go`)

```
startup → DetectCapability()
        → create KittyProvider or HalfBlockProvider
        → ui.SetGraphicsProvider(provider)

draw loop → provider.Reset()
          → app.Render()        // widgets call surface.Place()
          → screen.Show()
          → kittyProvider.Flush(tty)
```

### Texelation Server (pane management)

```
pane created → create RemoteGraphicsProvider(paneID, sender)
             → ui.SetGraphicsProvider(provider)

render cycle → provider.Reset()    → sends MsgImageReset
             → app.Render()        // surface.Place() sends MsgImagePlace
             → surface.Update()      sends MsgImageUpload (only when dirty)
```

### Texelation Client (renderer)

```
startup → DetectCapability() → local terminal capability
        → create ImageCache

message → ImageUpload: cache + decode
        → ImagePlace: add to placement list
        → ImageReset: clear placements
        → ImageDelete: free cache entry

render → composite cells
       → if non-Kitty: render half-block into cell buffer
       → screen.Show()
       → if Kitty: flush APC commands
```

## Migration from v1

The current `PlaceImage(ImagePlacement)` API is replaced. The `Image` widget and demo app are updated to use the surface API. The old `ImagePlacement` struct is removed; `ImageSurface` replaces it.

Breaking changes:
- `GraphicsProvider` interface methods change
- `KittyProvider` and `HalfBlockProvider` are rewritten
- `Image` widget uses surfaces instead of raw bytes

Since the current API was introduced in the same release (not yet merged to main), there are no external consumers to migrate.

## Future: Shared Memory

The `ImageSurface.Buffer()` API is designed for shared memory. When implemented:

1. `KittyProvider.CreateSurface()` allocates `Pix` via `mmap` instead of `make([]byte)`
2. `Update()` tells the terminal to read from shm (`a=t,t=s`) instead of inline base64
3. Zero-copy: app (or Cairo) renders directly into shared memory, terminal reads it

No API change required. The switch is internal to `KittyProvider`.

## File Structure

```
texelui/
  core/graphics.go                — GraphicsProvider + ImageSurface interfaces
  graphics/kitty.go               — KittyProvider (surfaces, APC encoding)
  graphics/halfblock.go           — HalfBlockProvider (surface → half-block art)
  graphics/detect.go              — env var detection (unchanged)
  runtime/runner.go               — Reset() + Flush() in draw loop
  widgets/image.go                — updated to use surfaces

texelation/
  protocol/image_messages.go      — ImageUpload/Place/Delete/Reset messages
  protocol/protocol.go            — new message type constants
  internal/runtime/server/        — RemoteGraphicsProvider, wiring
  client/image_cache.go           — client-side image cache
  internal/runtime/client/        — renderer integration, Kitty flush
```
