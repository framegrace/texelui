package textrender

import (
	"os"
	"testing"
)

func TestParseGhosttyConfig(t *testing.T) {
	config := `
# Ghostty config
font-size = 14
# font-family = NotThis
font-family = JetBrains Mono
theme = dark
`
	got := parseGhosttyFont(config)
	want := "JetBrains Mono"
	if got != want {
		t.Errorf("parseGhosttyFont = %q, want %q", got, want)
	}
}

func TestParseGhosttyConfig_Missing(t *testing.T) {
	config := `
# No font-family here
font-size = 14
theme = dark
`
	got := parseGhosttyFont(config)
	if got != "" {
		t.Errorf("parseGhosttyFont = %q, want empty string", got)
	}
}

func TestParseKittyConfig(t *testing.T) {
	config := `
# Kitty config
# font_family NotThis
font_size 13.0
font_family Fira Code
background #1e1e2e
`
	got := parseKittyFont(config)
	want := "Fira Code"
	if got != want {
		t.Errorf("parseKittyFont = %q, want %q", got, want)
	}
}

func TestParseKittyConfig_Missing(t *testing.T) {
	config := `
# No font_family here
font_size 13.0
background #1e1e2e
`
	got := parseKittyFont(config)
	if got != "" {
		t.Errorf("parseKittyFont = %q, want empty string", got)
	}
}

func TestDetectFont_Integration(t *testing.T) {
	path, err := DetectFont()
	if err != nil {
		t.Skipf("DetectFont() not available in this environment: %v", err)
	}
	if path == "" {
		t.Fatal("DetectFont() returned empty path")
	}
	t.Logf("Detected font: %s", path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("font file not accessible: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("font file is empty")
	}
}
