package render

import (
	"fmt"
	"strconv"
	"strings"

	"coralline/internal/conf"
	"coralline/internal/epoch"
	"coralline/internal/gitstate"
)

// Limit is a resolved rate-limit value for a gauge segment. An empty Pct hides
// the segment.
type Limit struct {
	Pct   string // percentage (raw string; may be synced high-water)
	Reset string // resets_at (ISO or epoch) for the countdown
}

// BurnResult is the precomputed burn projection (from package usage). State is
// "active", "idle", or "warming".
type BurnResult struct {
	State string
	Label string // "5h" or "7d" when active
	ETA   int64  // seconds until empty (when active)
	TTR   int64  // seconds until the window resets (when active)
}

// Data is everything the segments render from. Rate-limit and burn values are
// resolved upstream (limit-sync applied, burn estimated) so render stays pure.
type Data struct {
	Cwd    string
	Home   string
	Model  string
	CtxPct string
	TokIn  int64
	TokOut int64
	TokCR  int64
	TokCW  int64
	Effort string
	Git    gitstate.State
	Fh     Limit
	Wd     Limit
	Burn   BurnResult
	// BurnGuard mirrors bash's `[ -n fh_pct ] || [ -n wd_pct ]`: burn is hidden
	// entirely when neither input percentage was present.
	BurnGuard bool
	Now       int64
}

// builder produces a segment; ok=false hides it.
type builder func(c *conf.Config, d Data) (Segment, bool)

var builders = map[string]builder{
	"dir":     segDir,
	"git":     segGit,
	"model":   segModel,
	"effort":  segEffort,
	"ctx":     segCtx,
	"limit5h": segLimit5h,
	"limit7d": segLimit7d,
	"burn":    segBurn,
}

// ---- helpers ----

func roundPct(s string) int {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	v, _ := strconv.Atoi(fmt.Sprintf("%.0f", f))
	return v
}

func confInt(c *conf.Config, key string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(c.Get(key))); err == nil {
		return n
	}
	return def
}

// bar builds the gauge bar using the configured width/glyphs.
func bar(c *conf.Config, pct int) string {
	return Bar(pct, confInt(c, "VL_BAR_WIDTH", 5), c.Get("VL_BAR_FILL"), c.Get("VL_BAR_EMPTY"))
}

// pctColor resolves the threshold color spec for pct.
func pctColor(c *conf.Config, pct int) string {
	return PctFG(pct, confInt(c, "VL_WARN_PCT", 50), confInt(c, "VL_HOT_PCT", 75),
		c.Get("VL_FG_OK"), c.Get("VL_FG_WARN"), c.Get("VL_FG_HOT"))
}

// fmtCountdown mirrors statusline.sh fmt_countdown: "" when unparseable, "now"
// when expired, else the largest-unit d/h/m form.
func fmtCountdown(reset string, now int64) string {
	ep, ok := epoch.ToEpoch(reset)
	if !ok {
		return ""
	}
	diff := ep - now
	if diff <= 0 {
		return "now"
	}
	d := diff / 86400
	h := (diff % 86400) / 3600
	m := (diff % 3600) / 60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd%02dh", d, h)
	case h > 0:
		return fmt.Sprintf("%dh%02dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// fmtETA mirrors statusline.sh fmt_eta (seconds → d/h/m, no "now" case).
func fmtETA(s int64) string {
	if s < 0 {
		s = 0
	}
	d := s / 86400
	h := (s % 86400) / 3600
	m := (s % 3600) / 60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd%02dh", d, h)
	case h > 0:
		return fmt.Sprintf("%dh%02dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// trunc mirrors statusline.sh trunc: middle-truncate s to max visible chars with
// an ellipsis; max<=0 leaves s unchanged.
func trunc(s string, max int) string {
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	if max < 3 {
		return string(r[:max])
	}
	head := (max - 1) / 2
	tail := max - 1 - head
	return string(r[:head]) + "…" + string(r[len(r)-tail:])
}

// collapsePath mirrors seg_dir's path collapsing: replace a $HOME prefix with ~,
// then if the path has more than depth components, keep first/second/…/last.
// The split keeps empty fields (bash `set -- $short` with IFS=/ retains the
// leading empty field for an absolute path, so "/Users/demo/projects/coralline"
// has 5 fields with $1 empty → "/Users/…/coralline").
func collapsePath(cwd, home string, depth int) string {
	short := cwd
	if home != "" && strings.HasPrefix(cwd, home) {
		short = "~" + cwd[len(home):]
	}
	fields := strings.Split(short, "/")
	if len(fields) > depth {
		second := ""
		if len(fields) > 1 {
			second = fields[1]
		}
		return fields[0] + "/" + second + "/…/" + fields[len(fields)-1]
	}
	return short
}

// ---- segments ----

func segDir(c *conf.Config, d Data) (Segment, bool) {
	if d.Cwd == "" {
		return Segment{}, false
	}
	short := collapsePath(d.Cwd, d.Home, confInt(c, "VL_PATH_DEPTH", 4))
	text := Bold + FG(c.Get("VL_FG_TEXT")) + " " + short + " " + Norm
	return Segment{BG: c.Get("VL_BG_DIR"), Text: text}, true
}

func segGit(c *conf.Config, d Data) (Segment, bool) {
	if !d.Git.Present() || d.Git.Branch == "" {
		return Segment{}, false
	}
	bg := c.Get("VL_BG_GIT_OK")
	if d.Git.Dirty {
		bg = c.Get("VL_BG_GIT_DIRTY")
	}
	branch := trunc(d.Git.Branch, confInt(c, "VL_NAME_MAX", 0))
	text := Bold + FG(c.Get("VL_FG_TEXT")) + " ⎇ " + branch + d.Git.Marks + d.Git.AB + " " + Norm
	return Segment{BG: bg, Text: text}, true
}

func segModel(c *conf.Config, d Data) (Segment, bool) {
	if d.Model == "" {
		return Segment{}, false
	}
	name := strings.TrimPrefix(d.Model, "Claude ")
	text := Bold + FG(c.Get("VL_FG_TEXT")) + " ◆ " + name + " " + Norm
	return Segment{BG: c.Get("VL_BG_MODEL"), Text: text}, true
}

func segEffort(c *conf.Config, d Data) (Segment, bool) {
	if d.Effort == "" {
		return Segment{}, false
	}
	label := d.Effort
	if label == "medium" {
		label = "med"
	}
	text := FG(c.Get("VL_FG_TEXT")) + " ψ " + label + " "
	return Segment{BG: c.Get("VL_BG_EFFORT"), Text: text}, true
}

func segCtx(c *conf.Config, d Data) (Segment, bool) {
	if d.CtxPct == "" {
		return Segment{}, false
	}
	ci := roundPct(d.CtxPct)
	b := bar(c, ci)
	fgc := FG(pctColor(c, ci))
	detail := ""
	if c.Get("VL_CTX_TOKENS") != "0" { // bash default ${VL_CTX_TOKENS:-1}=1
		fgd := FG(c.Get("VL_FG_DIM"))
		detail = fgd + "↑" + FmtTok(d.TokIn) + " ↓" + FmtTok(d.TokOut) +
			" cr:" + FmtTok(d.TokCR) + " cw:" + FmtTok(d.TokCW) + " "
	}
	text := fgc + " ⬡ " + b + " " + strconv.Itoa(ci) + "% " + detail
	return Segment{BG: c.Get("VL_BG_CTX"), Text: text}, true
}

// segLimit is the shared limit gauge (statusline.sh seg_limit).
func segLimit(c *conf.Config, label string, lim Limit, bgKey string, now int64) (Segment, bool) {
	if lim.Pct == "" {
		return Segment{}, false
	}
	v := roundPct(lim.Pct)
	b := bar(c, v)
	fgc := FG(pctColor(c, v))
	rst := ""
	if cd := fmtCountdown(lim.Reset, now); cd != "" {
		rst = FG(c.Get("VL_FG_DIM")) + "↺ " + cd
	}
	text := fgc + " " + label + " " + b + " " + strconv.Itoa(v) + "% " + rst + " "
	return Segment{BG: c.Get(bgKey), Text: text}, true
}

func segLimit5h(c *conf.Config, d Data) (Segment, bool) {
	return segLimit(c, "5h", d.Fh, "VL_BG_5H", d.Now)
}

func segLimit7d(c *conf.Config, d Data) (Segment, bool) {
	return segLimit(c, "7d", d.Wd, "VL_BG_7D", d.Now)
}

func segBurn(c *conf.Config, d Data) (Segment, bool) {
	if !d.BurnGuard {
		return Segment{}, false
	}
	bg := c.Get("VL_BG_BURN")
	if bg == "" {
		bg = c.Get("VL_BG_5H")
	}
	glyph := c.Get("VL_BURN_GLYPH")

	if d.Burn.State != "active" {
		fgd := FG(c.Get("VL_FG_DIM"))
		if d.Burn.State == "warming" {
			return Segment{BG: bg, Text: fgd + " " + glyph + " … "}, true
		}
		// idle
		return Segment{BG: bg, Text: fgd + " " + glyph + " ✓ "}, true
	}

	// active
	var win int64 = 604800
	if d.Burn.Label == "5h" {
		win = 18000
	}
	if d.Burn.ETA > win {
		return Segment{BG: bg, Text: FG(c.Get("VL_FG_OK")) + " " + glyph + " ✓ "}, true
	}
	var col string
	switch {
	case d.Burn.ETA <= d.Burn.TTR:
		col = c.Get("VL_FG_HOT")
	case 10*d.Burn.TTR >= 8*d.Burn.ETA:
		col = c.Get("VL_FG_WARN")
	default:
		col = c.Get("VL_FG_OK")
	}
	text := FG(col) + " " + glyph + " " + d.Burn.Label + " ⇢ " + fmtETA(d.Burn.ETA) + " "
	return Segment{BG: bg, Text: text}, true
}
