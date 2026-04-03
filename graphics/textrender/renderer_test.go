package textrender

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/framegrace/texelui/core"
	"github.com/gdamore/tcell/v2"
)

func findTestFont(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("fc-match", "monospace", "--format=%{file}").Output()
	if err != nil {
		t.Skip("fc-match not available")
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		t.Skip("no monospace font found")
	}
	return path
}

func TestNewRenderer(t *testing.T) {
	fontPath := findTestFont(t)

	r, err := New(Config{FontPath: fontPath})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if r.cellW <= 0 {
		t.Errorf("cellW = %d, want > 0", r.cellW)
	}
	if r.cellH <= 0 {
		t.Errorf("cellH = %d, want > 0", r.cellH)
	}
}

func TestNewRenderer_WithExplicitSize(t *testing.T) {
	fontPath := findTestFont(t)

	r, err := New(Config{
		FontPath:   fontPath,
		CellWidth:  10,
		CellHeight: 20,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if r.cellW != 10 {
		t.Errorf("cellW = %d, want 10", r.cellW)
	}
	if r.cellH != 20 {
		t.Errorf("cellH = %d, want 20", r.cellH)
	}
}

func TestNewRenderer_BadPath(t *testing.T) {
	_, err := New(Config{FontPath: "/nonexistent/font.ttf"})
	if err == nil {
		t.Error("New() with bad path: expected error, got nil")
	}
}

func TestRender_BasicGrid(t *testing.T) {
	fontPath := findTestFont(t)

	r, err := New(Config{FontPath: fontPath})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	grid := [][]core.Cell{
		{
			{Ch: 'H', Style: tcell.StyleDefault},
			{Ch: 'i', Style: tcell.StyleDefault},
		},
		{
			{Ch: '!', Style: tcell.StyleDefault},
			{Ch: ' ', Style: tcell.StyleDefault},
		},
	}

	img := r.Render(grid)
	if img == nil {
		t.Fatal("Render() returned nil")
	}

	wantW := 2 * r.cellW
	wantH := 2 * r.cellH
	bounds := img.Bounds()
	if bounds.Dx() != wantW {
		t.Errorf("image width = %d, want %d", bounds.Dx(), wantW)
	}
	if bounds.Dy() != wantH {
		t.Errorf("image height = %d, want %d", bounds.Dy(), wantH)
	}
}
