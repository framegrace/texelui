# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TexelUI is a terminal UI library for building text-based applications in Go. It provides core primitives, widgets, layout managers, theming, and a runtime for standalone apps.

## CRITICAL: Git Workflow

**NEVER commit directly to main.** Always use feature branches and pull requests:

```bash
git checkout main && git pull
git checkout -b feature/my-feature   # or fix/, refactor/
# ... make changes ...
git add -A && git commit -m "Description"
git push -u origin feature/my-feature
# Create PR on GitHub to merge into main
```

This rule has no exceptions. All changes must go through PR review.

## Build and Development Commands

### Building
```bash
make build          # Build all packages
make demos          # Build CLI + demo binaries into ./bin
```

### Running
```bash
go run ./cmd/texelui-demo     # Widget showcase demo
go run ./cmd/texelui --help   # CLI server + bash adaptor
```

### Testing
```bash
make test           # Run all unit tests
make tidy           # Update go.mod dependencies
make clean          # Remove bin/
```

## Architecture

### Package Structure
- **core/** - Core interfaces and types: `App`, `Cell`, `Widget`, `ControlBus`, `UIManager`, `Painter`, storage interfaces
- **widgets/** - Widget implementations: Button, Input, Checkbox, ComboBox, TextArea, ColorPicker, Label, Form, StatusBar, etc.
- **primitives/** - Low-level visual components: TabBar, Grid, HCPlane, ScrollableList, ColorSwatch, etc.
- **layout/** - Layout managers: VBox, HBox
- **scroll/** - Scrolling infrastructure: ScrollPane, Viewport, PageNavigator, indicators
- **theme/** - Theme system: semantic colors, palettes, defaults, overrides
- **color/** - Color utilities: OKLCH color space support
- **runtime/** - Standalone app runner (used by apps running outside Texelation)
- **adapter/** - Texelation integration adapter (`UIApp`)
- **apps/** - Bundled apps: texeluicli (CLI server), texelui-demo

### Core Interfaces

**App** (`core/app.go`):
```go
type App interface {
    Run() error
    Stop()
    Resize(cols, rows int)
    Render() [][]Cell
    GetTitle() string
    HandleKey(ev *tcell.EventKey)
    SetRefreshNotifier(refreshChan chan<- bool)
}
```

**Widget** (`core/widget.go`):
- Embeddable UI components with bounds, focus, events
- BaseWidget provides default implementation

**UIManager** (`core/uimanager.go`):
- Manages widget tree, focus, events, dirty regions
- Tab/Shift-Tab focus traversal
- Click-to-focus support

### Theming

Widgets use semantic color keys from `theme/`:
- `bg.surface`, `bg.elevated`, `bg.primary`
- `text.primary`, `text.secondary`, `text.muted`
- `border.default`, `border.focus`
- `accent.primary`, `accent.secondary`

Theme config path: `~/.config/texelation/theme.json`

## Development Notes

- **Go Version**: 1.24.3
- **Formatting**: `gofmt` with tabs for indentation
- **Testing**: Table-driven tests in `_test.go` files
- **Commit Style**: Short present-tense (e.g., "Add checkbox validation"), subject < 60 chars

## Important Patterns

### Creating a Widget
```go
type MyWidget struct {
    widgets.BaseWidget
    // custom fields
}

func NewMyWidget(x, y, w, h int) *MyWidget {
    w := &MyWidget{}
    w.SetBounds(x, y, w, h)
    w.SetFocusable(true)
    return w
}

func (w *MyWidget) Draw(p core.Painter) {
    // drawing logic using painter
}

func (w *MyWidget) HandleKey(ev *tcell.EventKey) bool {
    // return true if handled
}
```

### Using UIManager
```go
ui := core.NewUIManager()
ui.Resize(cols, rows)

btn := widgets.NewButton(5, 5, 10, 1, "Click")
ui.AddWidget(btn)
ui.Focus(btn)

// In render loop
cells := ui.Render()
```

### Embedding in Texelation
```go
import (
    "github.com/framegrace/texelui/adapter"
    "github.com/framegrace/texelui/core"
)

ui := core.NewUIManager()
// ... add widgets ...
app := adapter.NewUIApp("My App", ui)
```

### Standalone App with Runtime
```go
// cmd/myapp/main.go
package main

import (
    "github.com/framegrace/texelui/core"
    "github.com/framegrace/texelui/runtime"
)

func main() {
    builder := func(_ []string) (core.App, error) {
        return NewMyApp(), nil
    }
    runtime.Run(builder)
}
```

## Widget Reference

| Widget | Package | Description |
|--------|---------|-------------|
| Button | widgets | Clickable with Enter/Space/mouse activation |
| Input | widgets | Single-line text entry with caret |
| TextArea | widgets | Multi-line editor with selection |
| Checkbox | widgets | Toggle with visual indicator |
| ComboBox | widgets | Dropdown selection |
| Label | widgets | Static text with alignment |
| Form | widgets | Form container with rows |
| Border | widgets | Decorator with border around child |
| Pane | widgets | Background surface with styling |
| StatusBar | widgets | Status display with sections |
| ColorPicker | widgets | Color selection (OKLCH, semantic, palette modes) |
| TabLayout | widgets | Tabbed interface container |
| ScrollPane | scroll | Scrollable viewport |

## Documentation

- `docs/TEXELUI_QUICKSTART.md` - Getting started guide
- `docs/TEXELUI_ARCHITECTURE_REVIEW.md` - Architecture analysis and roadmap
- `docs/TEXELUI_THEME.md` - Theme key reference

## Related Projects

- [Texelation](https://github.com/framegrace/texelation) - Text desktop environment that uses TexelUI
