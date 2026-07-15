package render

import "testing"

func TestFG(t *testing.T) {
	cases := map[string]string{
		"":           "",
		"173":        "\x1b[38;5;173m",
		"81,166,199": "\x1b[38;2;81;166;199m",
	}
	for spec, want := range cases {
		if got := FG(spec); got != want {
			t.Errorf("FG(%q) = %q, want %q", spec, got, want)
		}
	}
}

func TestBG(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"238":         "\x1b[48;5;238m",
		"137,180,250": "\x1b[48;2;137;180;250m",
	}
	for spec, want := range cases {
		if got := BG(spec); got != want {
			t.Errorf("BG(%q) = %q, want %q", spec, got, want)
		}
	}
}

// Bar fill uses integer rounding (pct*width+50)/100, capped at width.
func TestBar(t *testing.T) {
	const fill, empty = "F", "E"
	cases := []struct {
		pct, width int
		want       string
	}{
		{0, 5, "EEEEE"},
		{9, 5, "EEEEE"},   // (45+50)/100 = 0
		{10, 5, "FEEEE"},  // (50+50)/100 = 1
		{49, 5, "FFEEE"},  // (245+50)/100 = 2
		{50, 5, "FFFEE"},  // (250+50)/100 = 3
		{100, 5, "FFFFF"}, // (550)/100 = 5
		{150, 5, "FFFFF"}, // (800)/100 = 8, capped at 5
	}
	for _, c := range cases {
		if got := Bar(c.pct, c.width, fill, empty); got != c.want {
			t.Errorf("Bar(%d,%d) = %q, want %q", c.pct, c.width, got, c.want)
		}
	}
}

// Token abbreviation matches the bash fmt_tok integer math.
func TestFmtTok(t *testing.T) {
	cases := map[int64]string{
		0:       "0",
		999:     "999",
		1000:    "1.0k",
		1234:    "1.2k",
		999999:  "999.9k",
		1000000: "1.0M",
		1234567: "1.2M",
	}
	for n, want := range cases {
		if got := FmtTok(n); got != want {
			t.Errorf("FmtTok(%d) = %q, want %q", n, got, want)
		}
	}
}

// Threshold coloring: >= hot → hot, >= warn → warn, else ok.
func TestPctFG(t *testing.T) {
	ok, warn, hot := "OK", "WARN", "HOT"
	cases := []struct {
		pct  int
		want string
	}{
		{49, "OK"},
		{50, "WARN"},
		{79, "WARN"},
		{80, "HOT"},
	}
	for _, c := range cases {
		if got := PctFG(c.pct, 50, 80, ok, warn, hot); got != c.want {
			t.Errorf("PctFG(%d) = %q, want %q", c.pct, got, c.want)
		}
	}
}

func TestPill(t *testing.T) {
	segs := []Segment{{BG: "1", Text: "A"}, {BG: "2", Text: "B"}}
	got := Pill(segs, "L", "R", "S")
	want := Reset + "\x1b[38;5;1m" + "L" + // left cap in first bg color
		"\x1b[48;5;1m" + "A" + // seg 0 body
		"\x1b[48;5;2m" + "\x1b[38;5;1m" + "S" + // separator: next bg, cur fg
		"\x1b[48;5;2m" + "B" + // seg 1 body
		Reset + "\x1b[38;5;2m" + "R" + Reset // right cap in last bg color
	if got != want {
		t.Errorf("Pill mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestPillEmpty(t *testing.T) {
	if got := Pill(nil, "L", "R", "S"); got != "" {
		t.Errorf("Pill(nil) = %q, want empty", got)
	}
}
