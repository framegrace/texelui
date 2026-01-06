# TexelUI

A terminal UI library for building text-based applications in Go.

## Features

- **Core primitives**: App interface, Cell type, ControlBus, storage interfaces
- **Theme system**: Semantic colors + palettes, shared config path (`~/.config/texelation/theme.json`)
- **Widget library**: Button, Input, Checkbox, ComboBox, TextArea, ColorPicker, etc.
- **Layouts + scrolling**: VBox, HBox, ScrollPane, primitives
- **Texelation integration**: UIApp adapter for embedding in the desktop
- **Standalone tools**: TexelUI CLI + bash adaptor, demo app

## Installation

```bash
go get github.com/framegrace/texelui
```

## Quick Start

```bash
# Run the widget showcase demo
go run ./cmd/texelui-demo

# Use the CLI (server + bash adaptor)
go run ./cmd/texelui --help
```

## Embedding in Texelation

```go
ui := core.NewUIManager()
app := adapter.NewUIApp("My App", ui)
```

## License

AGPL-3.0-or-later

## Related Projects

- [Texelation](https://github.com/framegrace/texelation) - Text desktop environment using TexelUI
