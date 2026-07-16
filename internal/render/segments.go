package render

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	Cost     string // cost.total_cost_usd
	LinesAdd int64  // cost.total_lines_added
	LinesDel int64  // cost.total_lines_removed
	OutStyle string // output_style.name
	DurMs    int64  // cost.total_duration_ms

	StashCount int    // from gitstate
	GitRoot    string // from gitstate, for project segment

	NodeVersion   string // from runtime.DetectNode
	PythonVersion string // from runtime.DetectPython

	SegScan string // space-padded segment scan string, e.g. " dir git model ... "
}

// builder produces a segment; ok=false hides it.
type builder func(c *conf.Config, d Data) (Segment, bool)

var builders = map[string]builder{
	"dir":      segDir,
	"git":      segGit,
	"model":    segModel,
	"effort":   segEffort,
	"ctx":      segCtx,
	"limit5h":  segLimit5h,
	"limit7d":  segLimit7d,
	"burn":     segBurn,
	"cost":     segCost,
	"clock":    segClock,
	"lines":    segLines,
	"tokens":   segTokens,
	"style":    segStyle,
	"duration": segDuration,
	"stash":    segStash,
	"project":  segProject,
	"node":     segNode,
	"python":   segPython,
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

func segCost(c *conf.Config, d Data) (Segment, bool) {
	if d.Cost == "" || d.Cost == "0" {
		return Segment{}, false
	}
	decimals := confInt(c, "VL_COST_DECIMALS", 2)
	f, err := strconv.ParseFloat(d.Cost, 64)
	if err != nil {
		return Segment{}, false
	}
	formatted := fmt.Sprintf("$%.*f", decimals, f)
	text := FG(c.Get("VL_FG_TEXT")) + " " + formatted + " "
	return Segment{BG: c.Get("VL_BG_COST"), Text: text}, true
}

func segClock(c *conf.Config, d Data) (Segment, bool) {
	mode := c.Get("VL_CLOCK")
	if mode == "off" {
		return Segment{}, false
	}
	t := time.Unix(d.Now, 0)
	var formatted string
	if mode == "24h" {
		if c.Get("VL_CLOCK_SECONDS") == "1" {
			formatted = t.Format("15:04:05")
		} else {
			formatted = t.Format("15:04")
		}
	} else {
		if c.Get("VL_CLOCK_SECONDS") == "1" {
			formatted = t.Format("03:04:05 PM")
		} else {
			formatted = t.Format("03:04 PM")
		}
		formatted = strings.ToLower(formatted)
	}
	text := FG(c.Get("VL_FG_TEXT")) + " ⊙ " + formatted + " "
	return Segment{BG: c.Get("VL_BG_CLOCK"), Text: text}, true
}

func segLines(c *conf.Config, d Data) (Segment, bool) {
	if d.LinesAdd <= 0 && d.LinesDel <= 0 {
		return Segment{}, false
	}
	fgOk := FG(c.Get("VL_FG_OK"))
	fgHot := FG(c.Get("VL_FG_HOT"))
	text := " " + fgOk + "+" + strconv.FormatInt(d.LinesAdd, 10) + " " + fgHot + "-" + strconv.FormatInt(d.LinesDel, 10) + " "
	return Segment{BG: c.Get("VL_BG_LINES"), Text: text}, true
}

func segTokens(c *conf.Config, d Data) (Segment, bool) {
	if d.CtxPct == "" {
		return Segment{}, false
	}
	fgd := FG(c.Get("VL_FG_DIM"))
	bg := c.Get("VL_BG_TOKENS")
	if bg == "" {
		bg = c.Get("VL_BG_CTX")
	}
	text := fgd + " ↑" + FmtTok(d.TokIn) + " ↓" + FmtTok(d.TokOut) + " cr:" + FmtTok(d.TokCR) + " cw:" + FmtTok(d.TokCW) + " "
	return Segment{BG: bg, Text: text}, true
}

func segStyle(c *conf.Config, d Data) (Segment, bool) {
	if d.OutStyle == "" || d.OutStyle == "default" {
		return Segment{}, false
	}
	text := FG(c.Get("VL_FG_TEXT")) + " ✎ " + d.OutStyle + " "
	return Segment{BG: c.Get("VL_BG_STYLE"), Text: text}, true
}

// fmtDuration mirrors statusline.sh fmt_duration: ms → h/m/s.
func fmtDuration(ms int64) string {
	s := ms / 1000
	h := s / 3600
	m := (s % 3600) / 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

func segDuration(c *conf.Config, d Data) (Segment, bool) {
	if d.DurMs <= 0 {
		return Segment{}, false
	}
	text := FG(c.Get("VL_FG_TEXT")) + " ⧖ " + fmtDuration(d.DurMs) + " "
	return Segment{BG: c.Get("VL_BG_DURATION"), Text: text}, true
}

func segStash(c *conf.Config, d Data) (Segment, bool) {
	if !d.Git.Present() || d.StashCount <= 0 {
		return Segment{}, false
	}
	bg := c.Get("VL_BG_STASH")
	if bg == "" {
		bg = c.Get("VL_BG_GIT_OK")
	}
	text := FG(c.Get("VL_FG_TEXT")) + " ⚑ " + strconv.Itoa(d.StashCount) + " "
	return Segment{BG: bg, Text: text}, true
}

func segProject(c *conf.Config, d Data) (Segment, bool) {
	if d.GitRoot == "" {
		// Not in a git repo: fall back to dir if dir isn't already shown.
		if strings.Contains(d.SegScan, " dir ") {
			return Segment{}, false
		}
		return segDir(c, d)
	}
	name := filepath.Base(d.GitRoot)
	name = trunc(name, confInt(c, "VL_NAME_MAX", 0))
	bg := c.Get("VL_BG_PROJECT")
	if bg == "" {
		bg = c.Get("VL_BG_DIR")
	}
	text := Bold + FG(c.Get("VL_FG_TEXT")) + " ⬢ " + name + " " + Norm
	return Segment{BG: bg, Text: text}, true
}

func segNode(c *conf.Config, d Data) (Segment, bool) {
	if d.NodeVersion == "" {
		return Segment{}, false
	}
	bg := c.Get("VL_BG_NODE")
	if bg == "" {
		bg = c.Get("VL_BG_MODEL")
	}
	glyph := c.Get("VL_NODE_GLYPH")
	if glyph == "" {
		glyph = "\xee\x9c\x98" // U+E718 Nerd Font node
	}
	text := FG(c.Get("VL_FG_TEXT")) + " " + glyph + " " + d.NodeVersion + " "
	return Segment{BG: bg, Text: text}, true
}

func segPython(c *conf.Config, d Data) (Segment, bool) {
	if d.PythonVersion == "" {
		return Segment{}, false
	}
	bg := c.Get("VL_BG_PYTHON")
	if bg == "" {
		bg = c.Get("VL_BG_MODEL")
	}
	glyph := c.Get("VL_PY_GLYPH")
	if glyph == "" {
		glyph = "\xee\x9c\xbc" // U+E73C Nerd Font python
	}
	text := FG(c.Get("VL_FG_TEXT")) + " " + glyph + " " + d.PythonVersion + " "
	return Segment{BG: bg, Text: text}, true
}
