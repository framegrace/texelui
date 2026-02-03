// Copyright © 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: runtime/clipboard.go
// Summary: Standalone clipboard implementation for TexelUI runtime.

package runtime

import (
	"sync"

	"github.com/gdamore/tcell/v2"
)

// standaloneClipboard provides clipboard access for standalone mode.
// It uses tcell's screen clipboard for system clipboard integration.
type standaloneClipboard struct {
	mu     sync.RWMutex
	screen tcell.Screen
	// Fallback storage when screen is not available
	mime string
	data []byte
}

// newStandaloneClipboard creates a new standalone clipboard.
func newStandaloneClipboard(screen tcell.Screen) *standaloneClipboard {
	return &standaloneClipboard{screen: screen}
}

// SetClipboard stores data in the clipboard.
// For text/plain data, it also sets the system clipboard via tcell.
func (c *standaloneClipboard) SetClipboard(mime string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mime = mime
	c.data = append([]byte(nil), data...)

	// Set system clipboard for text data
	if c.screen != nil && mime == "text/plain" && len(data) > 0 {
		c.screen.SetClipboard(data)
	}
}

// GetClipboard retrieves the clipboard content.
func (c *standaloneClipboard) GetClipboard() (mime string, data []byte, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.data == nil {
		return "", nil, false
	}
	return c.mime, append([]byte(nil), c.data...), true
}
