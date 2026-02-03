// Copyright © 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: runtime/clipboard.go
// Summary: Standalone clipboard implementation for TexelUI runtime.

package runtime

import "sync"

// standaloneClipboard provides a simple in-memory clipboard for standalone mode.
// In the future, this could be enhanced to use platform-specific clipboard APIs.
type standaloneClipboard struct {
	mu   sync.RWMutex
	mime string
	data []byte
}

// newStandaloneClipboard creates a new standalone clipboard.
func newStandaloneClipboard() *standaloneClipboard {
	return &standaloneClipboard{}
}

// SetClipboard stores data in the clipboard.
func (c *standaloneClipboard) SetClipboard(mime string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mime = mime
	c.data = append([]byte(nil), data...) // Copy to avoid aliasing
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
