package textrender

import (
	"fmt"
	"io"
	"time"
)

// parseCellSizeResponse parses the terminal response to the \x1b[16t query.
// Expected format: \x1b[6;<height>;<width>t
func parseCellSizeResponse(data []byte) (width, height int, err error) {
	// Strip the ESC[ prefix and trailing 't'
	s := string(data)
	if len(s) < 2 || s[0] != '\x1b' || s[1] != '[' {
		return 0, 0, fmt.Errorf("textrender: malformed cell size response: missing ESC[")
	}
	s = s[2:] // strip ESC[

	var code, h, w int
	n, scanErr := fmt.Sscanf(s, "%d;%d;%dt", &code, &h, &w)
	if scanErr != nil || n != 3 {
		return 0, 0, fmt.Errorf("textrender: malformed cell size response: %w", scanErr)
	}
	if code != 6 {
		return 0, 0, fmt.Errorf("textrender: unexpected response code %d, want 6", code)
	}
	return w, h, nil
}

// QueryCellSize writes the \x1b[16t escape sequence to w, then reads the
// terminal's response from r (with the given timeout) and returns the pixel
// dimensions of a single character cell.
func QueryCellSize(w io.Writer, r io.Reader, timeout time.Duration) (width, height int, err error) {
	if _, err = fmt.Fprint(w, "\x1b[16t"); err != nil {
		return 0, 0, fmt.Errorf("textrender: writing cell-size query: %w", err)
	}

	type result struct {
		w, h int
		err  error
	}

	ch := make(chan result, 1)
	go func() {
		buf := make([]byte, 64)
		n, readErr := r.Read(buf)
		if readErr != nil {
			ch <- result{err: fmt.Errorf("textrender: reading cell-size response: %w", readErr)}
			return
		}
		cw, ch2, parseErr := parseCellSizeResponse(buf[:n])
		ch <- result{w: cw, h: ch2, err: parseErr}
	}()

	select {
	case res := <-ch:
		return res.w, res.h, res.err
	case <-time.After(timeout):
		return 0, 0, fmt.Errorf("textrender: timed out waiting for cell-size response")
	}
}
