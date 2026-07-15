// Package epoch parses reset timestamps to Unix epoch seconds, mirroring
// statusline.sh's to_epoch: it accepts an ISO 8601 timestamp (canonical UTC) or
// bare epoch seconds (with optional fractional part).
package epoch

import (
	"strconv"
	"strings"
	"time"
)

// isoLayouts are tried in order for a value containing 'T'.
var isoLayouts = []string{
	time.RFC3339Nano,                // with tz offset or Z, optional fraction
	time.RFC3339,                    // with tz offset or Z
	"2006-01-02T15:04:05.999999999", // no tz (treated as UTC below)
	"2006-01-02T15:04:05",           // no tz (treated as UTC below)
}

// ToEpoch converts t to Unix epoch seconds. ok is false when t is empty or
// unparseable.
func ToEpoch(t string) (epoch int64, ok bool) {
	if t == "" {
		return 0, false
	}
	if strings.Contains(t, "T") {
		for i, layout := range isoLayouts {
			// Layouts without a zone offset are parsed as UTC (indexes 2,3).
			if i >= 2 {
				if tm, err := time.ParseInLocation(layout, t, time.UTC); err == nil {
					return tm.Unix(), true
				}
				continue
			}
			if tm, err := time.Parse(layout, t); err == nil {
				return tm.Unix(), true
			}
		}
		return 0, false
	}
	// Bare epoch: drop any fractional part.
	s := t
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i]
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, true
	}
	return 0, false
}
