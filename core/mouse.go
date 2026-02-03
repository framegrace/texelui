// Copyright © 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: core/mouse.go
// Summary: Unified mouse handling interface for TexelUI apps.

package core

import "github.com/gdamore/tcell/v2"

// MouseHandler is implemented by apps that handle mouse events.
// This is the unified interface for mouse handling that works identically
// in both standalone mode (via texelui/runtime) and embedded mode (via texelation).
//
// Apps receive raw tcell.EventMouse events and handle all mouse logic internally,
// including selection, wheel scrolling, clicks, and drags.
//
// Example implementation:
//
//	func (a *MyApp) HandleMouse(ev *tcell.EventMouse) {
//	    x, y := ev.Position()
//	    buttons := ev.Buttons()
//
//	    // Handle wheel events
//	    if buttons&tcell.WheelUp != 0 {
//	        a.scrollUp()
//	    }
//
//	    // Handle button clicks
//	    if buttons&tcell.Button1 != 0 {
//	        a.handleLeftClick(x, y)
//	    }
//	}
type MouseHandler interface {
	HandleMouse(ev *tcell.EventMouse)
}
