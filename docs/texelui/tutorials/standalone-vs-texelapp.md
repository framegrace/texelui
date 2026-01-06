# Standalone vs TexelApp Mode

TexelUI applications can run in two different modes. This guide explains both and helps you choose the right one.

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    TexelUI Application                          │
│                                                                 │
│  ┌─────────────────┐              ┌─────────────────┐          │
│  │   UIManager     │              │   UIManager     │          │
│  │   + Widgets     │              │   + Widgets     │          │
│  └────────┬────────┘              └────────┬────────┘          │
│           │                                │                    │
│           ▼                                ▼                    │
│  ┌─────────────────┐              ┌─────────────────┐          │
│  │     UIApp       │              │     UIApp       │          │
│  │   (Adapter)     │              │   (Adapter)     │          │
│  └────────┬────────┘              └────────┬────────┘          │
│           │                                │                    │
├───────────┼────────────────────────────────┼────────────────────┤
│           ▼                                ▼                    │
│  ┌─────────────────┐              ┌─────────────────┐          │
│  │  Standalone     │              │  TexelApp Mode  │          │
│  │  (runtime)        │              │  (Texelation)   │          │
│  │                 │              │                 │          │
│  │  • Direct tcell │              │  • Managed pane │          │
│  │  • Single app   │              │  • Multi-app    │          │
│  │  • No server    │              │  • Full desktop │          │
│  └─────────────────┘              └─────────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

## Standalone Mode

Your application runs directly in the terminal, without the Texelation server.

### When to Use

- **Development and testing** - Quick iteration without server setup
- **Simple tools** - Single-purpose utilities
- **Learning** - Easier to understand the basics
- **Portability** - No Texelation infrastructure needed

### How It Works

```go
package main

import (
	"log"
	"github.com/framegrace/texelui/runtime"
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/adapter"
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/widgets"
)

func main() {
	// runtime.Run creates a tcell screen and runs the event loop
	err := runtime.Run(createApp)
	if err != nil {
		log.Fatal(err)
	}
}

func createApp(args []string) (core.App, error) {
	ui := core.NewUIManager()

	btn := widgets.NewButton(10, 5, 0, 0, "Hello!")
	ui.AddWidget(btn)
	ui.Focus(btn)

	return adapter.NewUIApp("Standalone App", ui), nil
}
```

### Architecture

```
┌─────────────────────────────────────────┐
│              Your Terminal              │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │          runtime.Run           │   │
│  │                                 │   │
│  │  1. Creates tcell.Screen       │   │
│  │  2. Enables mouse/paste        │   │
│  │  3. Calls your app factory     │   │
│  │  4. Runs event loop:           │   │
│  │     - PollEvent()              │   │
│  │     - Route to HandleKey/Mouse │   │
│  │     - Call Render()            │   │
│  │     - Update screen            │   │
│  │  5. Ctrl+C exits               │   │
│  │                                 │   │
│  └─────────────────────────────────┘   │
│                                         │
└─────────────────────────────────────────┘
```

### Building Standalone Apps

```bash
# Create binary
go build -o bin/my-app ./cmd/my-app

# Run directly
./bin/my-app

# Or via go run
go run ./cmd/my-app
```

### Registering with runtime

For reusable standalone apps, register with the runtime registry:

```go
// In init() or main
runtime.Register("my-app", func(args []string) (core.App, error) {
	return NewMyApp(args), nil
})
```

Then call from main:

```go
func main() {
	err := runtime.RunApp("my-app", os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
```

## TexelApp Mode

Your application runs inside the Texelation desktop environment as a managed pane.

### When to Use

- **Full integration** - Workspaces, effects, protocol features
- **Multi-app environment** - Run alongside other apps
- **Persistence** - Session restore, state management
- **Desktop features** - Status bar, launcher, pane management

### How It Works

```go
package myapp

import (
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/adapter"
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/widgets"
)

// New creates the TexelApp
func New() core.App {
	ui := core.NewUIManager()

	btn := widgets.NewButton(10, 5, 0, 0, "Hello!")
	ui.AddWidget(btn)
	ui.Focus(btn)

	app := adapter.NewUIApp("My App", ui)
	app.OnResize(func(w, h int) {
		// Handle pane resize
		btn.SetPosition(w/2-5, h/2)
	})

	return app
}
```

### Registration

Register your app in the server:

```go
// In cmd/texel-server/main.go
apps := map[string]func() core.App{
	"my-app": myapp.New,
	// ... other apps
}
```

### Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                    Texelation Desktop                          │
│                                                                │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐               │
│  │   Pane 1   │  │   Pane 2   │  │   Pane 3   │               │
│  │            │  │            │  │            │               │
│  │  Terminal  │  │  Your App  │  │    Help    │               │
│  │            │  │            │  │            │               │
│  └────────────┘  └────────────┘  └────────────┘               │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                    Desktop Manager                        │ │
│  │                                                          │ │
│  │  • Manages pane tree            • Handles effects        │ │
│  │  • Routes events                • Applies themes         │ │
│  │  • Broadcasts snapshots         • Manages sessions       │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│                        ▲       │                              │
│                        │       ▼                              │
│                 ┌──────────────────┐                          │
│                 │     Protocol     │                          │
│                 │  (Binary, Fast)  │                          │
│                 └────────┬─────────┘                          │
│                          │                                    │
├──────────────────────────┼────────────────────────────────────┤
│                          ▼                                    │
│                 ┌──────────────────┐                          │
│                 │  texel-client    │                          │
│                 │  (tcell render)  │                          │
│                 └──────────────────┘                          │
│                                                                │
│                     Your Terminal                              │
└────────────────────────────────────────────────────────────────┘
```

### Running in TexelApp Mode

```bash
# Start the server
make server

# In another terminal, start the client
make client

# Use Ctrl+Space to open launcher and start your app
```

## Comparison

| Feature | Standalone | TexelApp |
|---------|------------|----------|
| **Setup** | Simple | Requires server+client |
| **Startup** | Instant | Needs connection |
| **Other apps** | No | Yes, side-by-side |
| **Desktop features** | No | Full (effects, status bar) |
| **Session persistence** | No | Yes |
| **Resource usage** | Lower | Higher |
| **Development** | Faster iteration | Full integration testing |

## Supporting Both Modes

You can design your app to support both modes:

```go
package myapp

import (
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/adapter"
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/widgets"
)

// New creates the app (works for both modes)
func New() core.App {
	ui := core.NewUIManager()

	// Build UI
	buildUI(ui)

	app := adapter.NewUIApp("My App", ui)
	app.OnResize(func(w, h int) {
		handleResize(ui, w, h)
	})

	return app
}

func buildUI(ui *core.UIManager) {
	// Your UI setup
}

func handleResize(ui *core.UIManager, w, h int) {
	// Your resize logic
}
```

For standalone:
```go
// cmd/my-app/main.go
package main

import (
	"log"
	"texelation/apps/myapp"
	"github.com/framegrace/texelui/runtime"
)

func main() {
	err := runtime.Run(func(args []string) (core.App, error) {
		return myapp.New(), nil
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
}
```

For TexelApp:
```go
// Register in server
apps["my-app"] = myapp.New
```

## Best Practices

### 1. Always Handle Resize

```go
app.OnResize(func(w, h int) {
	// Reposition/resize widgets for new dimensions
})
```

### 2. Use Semantic Theme Colors

```go
// Good - adapts to theme
fg := tm.GetSemanticColor("text.primary")

// Bad - hardcoded
fg := tcell.ColorWhite
```

### 3. Test in Both Modes

During development:
1. Test standalone for fast iteration
2. Test in TexelApp for full integration

### 4. Handle Edge Cases

```go
app.OnResize(func(w, h int) {
	// Handle very small windows
	if w < 20 || h < 5 {
		// Minimal UI or error message
		return
	}
	// Normal layout
})
```

## What's Next?

- [Core Concepts](/texelui/core-concepts/README.md) - Understand the architecture
- [Integration Guide](/texelui/integration/README.md) - Deep dive into both modes
- [Building a Form](/texelui/tutorials/building-a-form.md) - Practice with a real example
