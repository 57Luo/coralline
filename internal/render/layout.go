package render

import (
	"strings"

	"coralline/internal/conf"
)

// Statusline renders the fixed multi-line pill layout. It walks VL_SEGMENTS,
// VL_SEGMENTS2, VL_SEGMENTS3 (one list per line, in order, capped at
// VL_MAX_LINES), builds each list's visible+implemented segments, and emits one
// pill row per non-empty list. A line whose segments are all hidden is omitted.
// Each emitted line ends with a newline, matching statusline.sh print_range.
func Statusline(c *conf.Config, d Data) string {
	lists := []string{
		c.Get("VL_SEGMENTS"),
		c.Get("VL_SEGMENTS2"),
		c.Get("VL_SEGMENTS3"),
	}
	maxLines := confInt(c, "VL_MAX_LINES", 3)

	capL := c.Get("VL_CAP_L")
	capR := c.Get("VL_CAP_R")
	sep := c.Get("VL_SEP")

	var out strings.Builder
	emitted := 0
	for _, list := range lists {
		if emitted >= maxLines {
			break
		}
		if strings.TrimSpace(list) == "" {
			continue
		}
		segs := buildLine(c, d, list)
		if len(segs) == 0 {
			continue // whole line hidden → omit
		}
		out.WriteString(Pill(segs, capL, capR, sep))
		out.WriteByte('\n')
		emitted++
	}
	return out.String()
}

// buildLine builds the visible segments named in a single space-separated list,
// in order. Unknown/unimplemented names are silently skipped.
func buildLine(c *conf.Config, d Data, list string) []Segment {
	var segs []Segment
	for _, name := range strings.Fields(list) {
		b, ok := builders[name]
		if !ok {
			continue // not implemented in this batch → silently skip
		}
		if seg, visible := b(c, d); visible {
			segs = append(segs, seg)
		}
	}
	return segs
}
