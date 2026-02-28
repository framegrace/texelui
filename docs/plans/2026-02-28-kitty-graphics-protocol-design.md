# Kitty Graphics Protocol for Image Widget

**Date:** 2026-02-28
**Status:** Approved
**Scope:** Standalone TexelUI apps (Phase 1), architected for texelation adaptation (Phase 2)

## Problem

The Image widget renders images using half-block Unicode characters (`U+2580`), producing ~2 vertical pixels per cell. This is functional but low fidelity. Modern terminals (Kitty, WezTerm, Ghostty) support the Kitty graphics protocol, which displays true raster images inline using APC escape sequences.

## Decision

Implement a `GraphicsProvider` interface in `core/` that abstracts image rendering capabilities. The standalone runtime injects a `KittyProvider` (or `HalfBlockProvider` fallback) at startup. The Image widget queries the provider and uses Kitty protocol when available, falling back to the current half-block rendering otherwise.

## Architecture

### Core Interface

```go
// core/graphics.go

type GraphicsCapability int

const (
    GraphicsNone      GraphicsCapability = iota
    GraphicsHalfBlock
    GraphicsKitty
)

type ImagePlacement struct {
    ID      uint32
    Rect    Rect
    ImgData []byte // raw PNG bytes
    ZIndex  int
}

type GraphicsProvider interface {
    Capability() GraphicsCapability
    PlaceImage(p ImagePlacement) error
    DeleteImage(id uint32)
    DeleteAll()
}
```

UIManager exposes `SetGraphicsProvider()`. The Painter exposes `GraphicsProvider()` so widgets can query it during `Draw()`.

### Kitty Protocol Details

- **Escape format:** `ESC_G<key=value,...>;<base64 payload>ESC\`
- **Image format:** `f=100` (PNG) — lets the terminal decode, avoids sending raw RGB
- **Transmission:** `t=d` (direct), base64-encoded, chunked at 4096 bytes (`m=1` for continuation, `m=0` for final)
- **Sizing:** `c=cols,r=rows` scales image to the widget's cell region
- **Positioning:** Cursor movement (`CSI row;col H`) before APC sequence
- **Suppression:** `q=2` suppresses error responses to avoid stdout noise

### Detection

At startup, the runtime sends a 1x1 pixel query:

```
ESC_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAAESCS\
```

Followed by `CSI c` (device attributes request). If the terminal responds with `ESC_Gi=31;OK ESC\` before the DA response, Kitty is supported. Timeout: 100ms. On failure, use `HalfBlockProvider`.

### File Structure

```
texelui/
  core/graphics.go             — GraphicsProvider interface + types
  graphics/detect.go           — Kitty capability detection
  graphics/kitty.go            — KittyProvider implementation
  graphics/halfblock.go        — HalfBlockProvider (no-op, signals half-block mode)
  runtime/runner.go            — detection, injection, post-Show flush
  widgets/image.go             — query provider, dual render path
```

### Image Widget Changes

```go
func (img *Image) Draw(p *core.Painter) {
    gp := p.GraphicsProvider()

    if gp != nil && gp.Capability() >= core.GraphicsKitty && img.valid {
        p.Fill(img.Rect, ' ', img.style)
        gp.PlaceImage(core.ImagePlacement{
            ID:      img.imageID,
            Rect:    img.Rect,
            ImgData: img.pngBytes,
            ZIndex:  -1,
        })
        return
    }

    img.drawHalfBlock(p)
}
```

- Keep raw PNG bytes (currently discarded after decode) for Kitty `f=100`
- Add `imageID uint32` assigned on first draw
- Add `lastRect Rect` to detect moves and trigger delete+re-place
- Extract current rendering into `drawHalfBlock()`

### Runtime Integration

The draw loop in `runtime/runner.go`:

```go
screen.Show()
// Post-show: flush queued Kitty placements
if gp, ok := graphicsProvider.(*graphics.KittyProvider); ok {
    gp.Flush(tty)
}
```

**Startup:**
1. `screen.Init()`
2. `tty := screen.Tty()`
3. `cap := graphics.DetectCapability(tty)` — Kitty query with timeout
4. Create `KittyProvider` or `HalfBlockProvider`
5. `ui.SetGraphicsProvider(provider)`

**Cleanup:** `provider.DeleteAll()` before `screen.Fini()`

**Resize:** `provider.DeleteAll()` on `EventResize` — widgets re-place on next draw.

### Timing: tcell Coordination

Kitty APC sequences are written directly to the terminal's file descriptor via `screen.Tty()`, bypassing tcell. The write happens *after* `screen.Show()` so tcell has already flushed its cell buffer. The Image widget fills its region with spaces so tcell clears the area before the Kitty image is placed on top.

### Fallback Behavior

```
Startup → DetectCapability(tty)
  ├─ Kitty supported → KittyProvider (APC sequences)
  └─ Not supported  → HalfBlockProvider
                       Image widget uses drawHalfBlock() (current behavior)
```

No user configuration needed. Auto-detection is transparent.

## Future: Texelation Adaptation (Phase 2)

The `GraphicsProvider` interface maps to the client/server model:

- **Server side:** `RemoteGraphicsProvider` serializes `ImagePlacement` into a new protocol message (`MsgImagePlacement{ID, Rect, PNGData}`)
- **Client side:** Receives the message, uses a local `KittyProvider` to write APC sequences to the real terminal
- **Texelterm:** Parser gains `StateAPC` handler to capture Kitty sequences from guest programs, converting them into `ImagePlacement` calls

The interface stays the same — only implementations change per context.
