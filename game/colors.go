package game

import (
	"fmt"
	"strconv"
	"strings"
)

// Colors styles text with SGR codes written directly: inside WebAssembly
// there's no terminal for lipgloss to detect, so it would draw no color.
type Colors struct {
	accent, good, bad, dim, key, blockFg string
}

// DefaultColors are the terminal's own palette (ANSI magenta, green, red,
// bright black and cyan), so a standalone game matches the terminal theme.
func DefaultColors() Colors {
	return FromPalette(map[string]string{
		"purple": "5", "green": "2", "red": "1", "comment": "8", "cyan": "6", "background": "0",
	})
}

// FromPalette uses a Concord theme palette (its "purple", "green", "red",
// "comment", "cyan" and "background"): #rrggbb colors or ANSI numbers.
// Missing entries fall back to the terminal's palette.
func FromPalette(p map[string]string) Colors {
	pick := func(name, fallback string) string {
		if v := p[name]; colorCode(v, false) != "" {
			return v
		}
		return fallback
	}
	return Colors{
		accent:  fg(pick("purple", "5"), true),
		good:    fg(pick("green", "2"), true),
		bad:     fg(pick("red", "1"), true),
		dim:     fg(pick("comment", "8"), false),
		key:     fg(pick("cyan", "6"), true),
		blockFg: fg(pick("background", "0"), true) + bg(pick("green", "2")),
	}
}

// fg is the SGR code for a foreground color, or "" if it can't be read.
func fg(c string, bold bool) string {
	code := colorCode(c, false)
	if code == "" {
		return ""
	}
	if bold {
		return "\x1b[1;" + code + "m"
	}
	return "\x1b[" + code + "m"
}

func bg(c string) string {
	if code := colorCode(c, true); code != "" {
		return "\x1b[" + code + "m"
	}
	return ""
}

// colorCode turns "#rrggbb" or an ANSI number into SGR parameters.
func colorCode(c string, background bool) string {
	base, extended := 30, "38"
	if background {
		base, extended = 40, "48"
	}
	if strings.HasPrefix(c, "#") && len(c) == 7 {
		var r, g, b int
		if _, err := fmt.Sscanf(c, "#%02x%02x%02x", &r, &g, &b); err == nil {
			return fmt.Sprintf("%s;2;%d;%d;%d", extended, r, g, b)
		}
		return ""
	}
	n, err := strconv.Atoi(c)
	switch {
	case err != nil || n < 0 || n > 255:
		return ""
	case n < 8:
		return strconv.Itoa(base + n)
	case n < 16:
		return strconv.Itoa(base + 60 + n - 8)
	}
	return fmt.Sprintf("%s;5;%d", extended, n)
}

func wrap(code, s string) string {
	if code == "" {
		return s
	}
	return code + s + "\x1b[0m"
}

func (c Colors) Accent(s string) string    { return wrap(c.accent, s) }
func (c Colors) Good(s string) string      { return wrap(c.good, s) }
func (c Colors) Bad(s string) string       { return wrap(c.bad, s) }
func (c Colors) Dim(s string) string       { return wrap(c.dim, s) }
func (c Colors) Key(s string) string       { return wrap(c.key, s) }
func (c Colors) GoodBlock(s string) string { return wrap(c.blockFg, s) }
