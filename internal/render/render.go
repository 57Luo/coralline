// Package render holds the ANSI primitives, bar/token formatters, pill assembly,
// and the fixed multi-line layout. The primitives are faithful ports of
// statusline.sh's fg/bg, make_bar, fmt_tok, pct_fg, and print_range (pill path).
package render

import (
	"strconv"
	"strings"
)

// ANSI control sequences, matching statusline.sh's R/BOLD/NORM.
const (
	Reset = "\x1b[0m"
	Bold  = "\x1b[1m"
	Norm  = "\x1b[22m"
)

// FG returns the foreground SGR escape for a color spec: a 256-color index, an
// "R,G,B" true-color triple, or "" (which yields "" to inherit the color).
func FG(spec string) string { return colorEscape(spec, "38") }

// BG returns the background SGR escape for a color spec (see FG).
func BG(spec string) string { return colorEscape(spec, "48") }

func colorEscape(spec, layer string) string {
	if spec == "" {
		return ""
	}
	if strings.Contains(spec, ",") {
		parts := strings.SplitN(spec, ",", 3)
		if len(parts) == 3 {
			return "\x1b[" + layer + ";2;" + parts[0] + ";" + parts[1] + ";" + parts[2] + "m"
		}
		return ""
	}
	return "\x1b[" + layer + ";5;" + spec + "m"
}

// Bar renders a gauge bar: filled = (pct*width+50)/100 (integer), capped at
// width. Cells use fill/empty glyphs.
func Bar(pct, width int, fill, empty string) string {
	filled := (pct*width + 50) / 100
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	var b strings.Builder
	for i := 0; i < filled; i++ {
		b.WriteString(fill)
	}
	for i := filled; i < width; i++ {
		b.WriteString(empty)
	}
	return b.String()
}

// FmtTok abbreviates a token count with integer math: >=1e6 → "<n>.<d>M",
// >=1e3 → "<n>.<d>k", else the bare integer.
func FmtTok(n int64) string {
	switch {
	case n >= 1000000:
		return strconv.FormatInt(n/1000000, 10) + "." + strconv.FormatInt((n%1000000)/100000, 10) + "M"
	case n >= 1000:
		return strconv.FormatInt(n/1000, 10) + "." + strconv.FormatInt((n%1000)/100, 10) + "k"
	default:
		return strconv.FormatInt(n, 10)
	}
}

// PctFG selects a color spec by threshold: pct>=hot → hot, pct>=warn → warn,
// else ok.
func PctFG(pct, warn, hot int, ok, warnColor, hotColor string) string {
	switch {
	case pct >= hot:
		return hotColor
	case pct >= warn:
		return warnColor
	default:
		return ok
	}
}

// Segment is one pill: a background color spec and its already-composed text
// (including any inline foreground escapes).
type Segment struct {
	BG   string
	Text string
}

// Pill assembles segments into one pill-style row (statusline.sh print_range,
// pill branch): a left cap in the first segment's color, each segment's body on
// its background, separators drawn in the current segment color over the next
// background, and a right cap in the last segment's color. Returns "" for no
// segments.
func Pill(segs []Segment, capL, capR, sep string) string {
	if len(segs) == 0 {
		return ""
	}
	var b strings.Builder
	last := len(segs) - 1

	// fg(SEG_BGS[0]); out="${R}${_FG}${VL_CAP_L}"
	b.WriteString(Reset)
	b.WriteString(FG(segs[0].BG))
	b.WriteString(capL)

	for i, s := range segs {
		// bg(SEG_BGS[i]); out+="${_BG}${SEG_TXT[i]}"
		b.WriteString(BG(s.BG))
		b.WriteString(s.Text)
		if i < last {
			// bg(next); fg(cur); out+="${_BG}${_FG}${VL_SEP}"
			b.WriteString(BG(segs[i+1].BG))
			b.WriteString(FG(s.BG))
			b.WriteString(sep)
		}
	}

	// fg(SEG_BGS[last]); out+="${R}${_FG}${VL_CAP_R}${R}"
	b.WriteString(Reset)
	b.WriteString(FG(segs[last].BG))
	b.WriteString(capR)
	b.WriteString(Reset)
	return b.String()
}
