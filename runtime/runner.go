// Copyright © 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: runtime/runner.go
// Summary: Runtime runner for TexelUI apps without Texelation.

package runtime

import (
	"fmt"
	"sync"

	"github.com/framegrace/texelui/adapter"
	"github.com/framegrace/texelui/core"
	"github.com/framegrace/texelui/graphics"
	"github.com/framegrace/texelui/theme"
	"github.com/gdamore/tcell/v2"
)

// Builder constructs a core.App, optionally using CLI args.
type Builder func(args []string) (core.App, error)

// Options controls the runtime runner behavior.
type Options struct {
	ExitKey      tcell.Key
	DisableMouse bool
	OnInit       func(screen tcell.Screen)
	OnExit       func()
}

var (
	screenFactory = tcell.NewScreen
	registryMu    sync.RWMutex
	registry      = map[string]Builder{}

	exitMu     sync.Mutex
	activeExit chan struct{}
)

// Register adds a builder to the runtime registry.
func Register(name string, builder Builder) {
	if name == "" || builder == nil {
		return
	}
	registryMu.Lock()
	registry[name] = builder
	registryMu.Unlock()
}

// RunApp runs a registered app by name.
func RunApp(name string, args []string) error {
	registryMu.RLock()
	builder := registry[name]
	registryMu.RUnlock()
	if builder == nil {
		return fmt.Errorf("runtime: unknown app %q", name)
	}
	return RunWithOptions(builder, Options{}, args...)
}

// Run runs a core.App builder in a terminal session.
func Run(builder Builder, args ...string) error {
	return RunWithOptions(builder, Options{}, args...)
}

// RunWithOptions runs a core.App builder with custom options.
func RunWithOptions(builder Builder, opts Options, args ...string) error {
	if builder == nil {
		return fmt.Errorf("runtime: nil builder")
	}
	app, err := builder(args)
	if err != nil {
		return err
	}
	return runApp(app, opts)
}

// RunUI runs a UIManager directly in a terminal session.
func RunUI(ui *core.UIManager) error {
	return RunUIWithOptions(ui, Options{})
}

// RunUIWithOptions runs a UIManager with custom options.
func RunUIWithOptions(ui *core.UIManager, opts Options) error {
	app := adapter.NewUIApp("", ui)
	return runApp(app, opts)
}

// RequestExit signals the active runner (if any) to exit.
func RequestExit() {
	exitMu.Lock()
	ch := activeExit
	exitMu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}

// SetScreenFactory overrides the screen factory used by the runner.
func SetScreenFactory(factory func() (tcell.Screen, error)) {
	if factory == nil {
		screenFactory = tcell.NewScreen
		return
	}
	screenFactory = factory
}

func normalizeOptions(opts Options) Options {
	if opts.ExitKey == 0 {
		opts.ExitKey = tcell.KeyEscape
	}
	return opts
}

func runApp(app core.App, opts Options) error {
	opts = normalizeOptions(opts)

	exitMu.Lock()
	activeExit = make(chan struct{}, 1)
	exitMu.Unlock()
	defer func() {
		exitMu.Lock()
		activeExit = nil
		exitMu.Unlock()
	}()

	screen, err := screenFactory()
	if err != nil {
		return fmt.Errorf("init screen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("screen init: %w", err)
	}
	defer screen.Fini()

	// Provide clipboard service to apps that support it (after screen init)
	clipboard := newStandaloneClipboard(screen)
	if ca, ok := app.(core.ClipboardAware); ok {
		ca.SetClipboardService(clipboard)
	}

	if opts.OnInit != nil {
		opts.OnInit(screen)
	}
	if !opts.DisableMouse {
		screen.EnableMouse(tcell.MouseMotionEvents)
		defer screen.DisableMouse()
	}
	screen.EnablePaste()

	_ = theme.Get()
	if err := theme.GetLoadError(); err != nil {
		return fmt.Errorf("theme: %w", err)
	}

	// Detect graphics capability
	var graphicsProvider core.GraphicsProvider
	if tty, ok := screen.Tty(); ok {
		gcap := graphics.DetectCapability(tty)
		if gcap == core.GraphicsKitty {
			graphicsProvider = graphics.NewKittyProvider()
		} else {
			graphicsProvider = graphics.NewHalfBlockProvider()
		}
	} else {
		graphicsProvider = graphics.NewHalfBlockProvider()
	}

	// Inject into UIManager if the app supports it
	if ua, ok := app.(interface{ UI() *core.UIManager }); ok {
		ua.UI().SetGraphicsProvider(graphicsProvider)
	}
	defer func() {
		if kp, ok := graphicsProvider.(*graphics.KittyProvider); ok {
			kp.DeleteAll()
			if tty, hasTty := screen.Tty(); hasTty {
				_ = kp.Flush(tty)
			}
		}
	}()

	width, height := screen.Size()
	app.Resize(width, height)
	refreshCh := make(chan bool, 1)
	app.SetRefreshNotifier(refreshCh)

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

	draw()

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()
	defer app.Stop()

	go func() {
		for range refreshCh {
			screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
	}()

	var pasteBuffer []byte
	var inPaste bool

	for {
		select {
		case err := <-runErr:
			if opts.OnExit != nil {
				opts.OnExit()
			}
			return err
		case <-activeExit:
			if opts.OnExit != nil {
				opts.OnExit()
			}
			return nil
		default:
		}

		ev := screen.PollEvent()
		switch tev := ev.(type) {
		case *tcell.EventInterrupt:
			draw()
		case *tcell.EventResize:
			w, h := tev.Size()
			graphicsProvider.DeleteAll()
			app.Resize(w, h)
			draw()
		case *tcell.EventPaste:
			if tev.Start() {
				inPaste = true
				pasteBuffer = nil
			} else if tev.End() {
				inPaste = false
				if ph, ok := app.(interface{ HandlePaste([]byte) }); ok && len(pasteBuffer) > 0 {
					ph.HandlePaste(pasteBuffer)
					draw()
				}
				pasteBuffer = nil
			}
		case *tcell.EventKey:
			// Only check ExitKey - don't hardcode Ctrl+C (apps like terminals need it)
			if opts.ExitKey != tcell.Key(-1) && tev.Key() == opts.ExitKey {
				if opts.OnExit != nil {
					opts.OnExit()
				}
				return nil
			}
			if inPaste {
				if tev.Key() == tcell.KeyRune {
					pasteBuffer = append(pasteBuffer, []byte(string(tev.Rune()))...)
				} else if tev.Key() == tcell.KeyEnter || tev.Key() == 10 {
					pasteBuffer = append(pasteBuffer, '\n')
				}
			} else {
				app.HandleKey(tev)
				draw()
			}
		case *tcell.EventMouse:
			if mh, ok := app.(interface{ HandleMouse(*tcell.EventMouse) }); ok {
				mh.HandleMouse(tev)
				draw()
			}
		}
	}
}
