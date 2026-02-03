// Copyright © 2025 Texelation contributors
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// File: core/clipboard.go
// Summary: Clipboard service interface for TexelUI apps.

package core

// ClipboardService provides clipboard access for apps.
// In standalone mode, this is provided by the runtime.
// In embedded mode (texelation), the desktop provides this service.
//
// Apps use this to copy/paste content, typically for text selection.
//
// Example usage:
//
//	func (a *MyApp) copySelection() {
//	    text := a.getSelectedText()
//	    if a.clipboard != nil {
//	        a.clipboard.SetClipboard("text/plain", []byte(text))
//	    }
//	}
type ClipboardService interface {
	// SetClipboard stores data in the clipboard with the given MIME type.
	// Common MIME types: "text/plain", "text/html"
	SetClipboard(mime string, data []byte)

	// GetClipboard retrieves the current clipboard content.
	// Returns the MIME type, data, and whether clipboard has content.
	GetClipboard() (mime string, data []byte, ok bool)
}

// ClipboardAware is implemented by apps that can receive a clipboard service.
// The runtime or desktop calls this during app initialization.
type ClipboardAware interface {
	SetClipboardService(clipboard ClipboardService)
}
