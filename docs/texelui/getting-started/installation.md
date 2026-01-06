# Installation

This guide covers how to install and build TexelUI as a standalone library and toolkit.

## Prerequisites

### Required
- **Go 1.24.3** or later
- **Git** for cloning the repository
- **Make** for building (optional but recommended)

### Recommended
- A modern terminal with 256-color support
- Terminal size of at least 80x24 characters

## Installation Steps

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/texelui.git
cd texelui
```

### 2. Build the Project

Using the Makefile (recommended):
```bash
# Build all packages
make build

# Build the TexelUI CLI + demo binaries
make demos
```

Using Go directly:
```bash
# Build the TexelUI CLI
go build -o bin/texelui ./cmd/texelui

# Build the demo app
go build -o bin/texelui-demo ./cmd/texelui-demo
```

### 3. Verify Installation

```bash
# Run the demo to verify everything works
./bin/texelui-demo
```

You should see a tabbed interface. Press `Ctrl+C` to exit.

## Project Structure

After building, you'll have these relevant binaries:

```
bin/
├── texelui          # TexelUI CLI server/client
└── texelui-demo     # TexelUI widget showcase (standalone)
```

## TexelUI Source Location

TexelUI source code is organized in top-level packages:

```
core/        # Core interfaces and UIManager
widgets/     # Built-in widgets
primitives/  # Reusable building blocks
layout/      # Layout managers
adapter/     # core.App adapter
scroll/      # ScrollPane + scroll helpers
theme/       # Theme loading + palettes
color/       # Color utilities (OKLCH, etc.)
runtime/     # Runtime runner
apps/        # Demo apps + texeluicli
cmd/         # CLI + demo entry points
```

## Development Setup

### IDE Configuration

For VS Code, add these settings for Go development:

```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.formatTool": "gofmt"
}
```

### Running Tests

```bash
# Run all TexelUI tests
make test

# Run with verbose output
go test -v ./...
```

## Troubleshooting

### Build Errors

**"package github.com/framegrace/texelui/... not found"**
- Ensure you're in the texelui root directory
- Run `go mod tidy` to update dependencies

**"tcell: terminal does not support required features"**
- Your terminal may not support the required escape sequences
- Try a different terminal (iTerm2, Alacritty, Kitty, etc.)

### Runtime Issues

**Colors look wrong**
- Ensure your terminal supports 256 colors
- Check `TERM` environment variable (should be `xterm-256color` or similar)
- The theme system defaults to Catppuccin Mocha palette

**Mouse doesn't work**
- Ensure your terminal supports mouse events
- Some terminals require enabling mouse mode in settings

## Next Steps

- Continue to [Quickstart](/texelui/getting-started/quickstart.md) to build your first app
- Or jump to [Hello World](/texelui/getting-started/hello-world.md) for a minimal example

## Related Documentation

- [CLAUDE.md](/CLAUDE.md) - Build commands and project overview
- [Makefile](/Makefile) - All available make targets
