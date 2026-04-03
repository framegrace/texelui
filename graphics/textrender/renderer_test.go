package textrender

import (
	"image/png"
	"os"
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

func TestRenderBasicGrid(t *testing.T) {
	fontPath := findTestFont(t)

	r, err := New(Config{FontPath: fontPath})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const cols, rows = 30, 5
	darkBg := tcell.NewRGBColor(0x1e, 0x1e, 0x2e)
	lightGray := tcell.NewRGBColor(0xcc, 0xcc, 0xcc)
	green := tcell.NewRGBColor(0x00, 0xe6, 0x00)
	white := tcell.ColorWhite

	cycleColors := []tcell.Color{
		tcell.ColorRed,
		tcell.ColorGreen,
		tcell.ColorBlue,
		tcell.ColorYellow,
		tcell.ColorFuchsia,
	}

	// Helper: build a blank row of spaces with lightGray on darkBg.
	blankRow := func() []core.Cell {
		row := make([]core.Cell, cols)
		for i := range row {
			row[i] = core.Cell{
				Ch:    ' ',
				Style: tcell.StyleDefault.Foreground(lightGray).Background(darkBg),
			}
		}
		return row
	}

	grid := make([][]core.Cell, rows)
	for i := range grid {
		grid[i] = blankRow()
	}

	// Row 0: "Hello, World!" in green on dark bg.
	msg0 := "Hello, World!"
	for i, ch := range msg0 {
		grid[0][i] = core.Cell{
			Ch:    ch,
			Style: tcell.StyleDefault.Foreground(green).Background(darkBg),
		}
	}

	// Row 1: '#' chars cycling through red/green/blue/yellow/magenta on dark bg.
	for i := 0; i < cols; i++ {
		grid[1][i] = core.Cell{
			Ch:    '#',
			Style: tcell.StyleDefault.Foreground(cycleColors[i%len(cycleColors)]).Background(darkBg),
		}
	}

	// Row 2: "BOLD TEXT" in white, bold attribute.
	msg2 := "BOLD TEXT"
	for i, ch := range msg2 {
		grid[2][i] = core.Cell{
			Ch:    ch,
			Style: tcell.StyleDefault.Foreground(white).Background(darkBg).Bold(true),
		}
	}

	// Row 3: "underlined" in light gray, underline attribute.
	msg3 := "underlined"
	for i, ch := range msg3 {
		grid[3][i] = core.Cell{
			Ch:    ch,
			Style: tcell.StyleDefault.Foreground(lightGray).Background(darkBg).Underline(true),
		}
	}

	// Row 4: "reversed" in light gray, reverse attribute.
	msg4 := "reversed"
	for i, ch := range msg4 {
		grid[4][i] = core.Cell{
			Ch:    ch,
			Style: tcell.StyleDefault.Foreground(lightGray).Background(darkBg).Reverse(true),
		}
	}

	img := r.Render(grid)
	if img == nil {
		t.Fatal("Render() returned nil")
	}

	bounds := img.Bounds()
	if bounds.Dx() <= 100 {
		t.Errorf("image width = %d, want > 100", bounds.Dx())
	}
	if bounds.Dy() <= 0 {
		t.Errorf("image height = %d, want > 0", bounds.Dy())
	}

	outDir := "testdata"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", outDir, err)
	}
	outPath := outDir + "/output.png"
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("os.Create(%q): %v", outPath, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	t.Logf("PNG written to %s", outPath)
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
