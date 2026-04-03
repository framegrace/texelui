package textrender

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func parseGhosttyFont(config string) string {
	scanner := bufio.NewScanner(strings.NewReader(config))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key == "font-family" {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

func parseKittyFont(config string) string {
	scanner := bufio.NewScanner(strings.NewReader(config))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == "font_family" {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

// ResolveFontPath runs fc-match to find the file path for the given font family.
// Returns os.ErrNotExist if fc-match returns no path.
func ResolveFontPath(family string) (string, error) {
	out, err := exec.Command("fc-match", family, "--format=%{file}").Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", os.ErrNotExist
	}
	return path, nil
}

// DetectFont tries to find a configured font from Ghostty or Kitty configs,
// resolving to a font file path via fc-match. Returns os.ErrNotExist if
// neither config yields a usable font.
func DetectFont() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configs := []struct {
		path  string
		parse func(string) string
	}{
		{filepath.Join(home, ".config", "ghostty", "config"), parseGhosttyFont},
		{filepath.Join(home, ".config", "kitty", "kitty.conf"), parseKittyFont},
	}

	for _, c := range configs {
		data, err := os.ReadFile(c.path)
		if err != nil {
			continue
		}
		family := c.parse(string(data))
		if family == "" {
			continue
		}
		path, err := ResolveFontPath(family)
		if err != nil {
			continue
		}
		return path, nil
	}

	return "", os.ErrNotExist
}
