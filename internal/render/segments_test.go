package render

import (
	"strings"
	"testing"

	"coralline/internal/conf"
	"coralline/internal/gitstate"
)

// burn warming is covered by the golden test; here we cover idle and active.
func TestSegBurnIdle(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segBurn(c, Data{BurnGuard: true, Burn: BurnResult{State: "idle"}})
	if !ok {
		t.Fatal("idle burn should be visible")
	}
	want := FG(c.Get("VL_FG_DIM")) + " " + c.Get("VL_BURN_GLYPH") + " ✓ "
	if seg.Text != want {
		t.Errorf("idle text = %q, want %q", seg.Text, want)
	}
}

func TestSegBurnActiveHot(t *testing.T) {
	c := conf.Defaults()
	// eta <= ttr → hot; eta (3600) < 5h window (18000) so not the ✓ shortcut.
	seg, ok := segBurn(c, Data{BurnGuard: true,
		Burn: BurnResult{State: "active", Label: "5h", ETA: 3600, TTR: 7200}})
	if !ok {
		t.Fatal("active burn should be visible")
	}
	want := FG(c.Get("VL_FG_HOT")) + " " + c.Get("VL_BURN_GLYPH") + " 5h ⇢ 1h00m "
	if seg.Text != want {
		t.Errorf("active-hot text = %q, want %q", seg.Text, want)
	}
}

func TestSegBurnActiveHealthyCheck(t *testing.T) {
	c := conf.Defaults()
	// eta beyond the window → ✓ in OK color.
	seg, _ := segBurn(c, Data{BurnGuard: true,
		Burn: BurnResult{State: "active", Label: "5h", ETA: 20000, TTR: 100}})
	want := FG(c.Get("VL_FG_OK")) + " " + c.Get("VL_BURN_GLYPH") + " ✓ "
	if seg.Text != want {
		t.Errorf("active-healthy text = %q, want %q", seg.Text, want)
	}
}

func TestSegBurnHiddenWithoutGuard(t *testing.T) {
	if _, ok := segBurn(conf.Defaults(), Data{BurnGuard: false, Burn: BurnResult{State: "warming"}}); ok {
		t.Error("burn should be hidden when neither limit percentage is present")
	}
}

// collapsePath must keep the leading empty field of an absolute path so it
// collapses exactly like bash `set -- $short` (verified against statusline.sh:
// /Users/demo/projects/coralline → /Users/…/coralline).
func TestCollapsePath(t *testing.T) {
	cases := []struct {
		cwd, home string
		depth     int
		want      string
	}{
		{"/Users/demo/projects/coralline", "", 4, "/Users/…/coralline"},   // 5 fields incl. leading empty
		{"/home/dev/coralline-nogit", "", 4, "/home/dev/coralline-nogit"}, // 4 fields → unchanged
		{"/c/Users/me/proj/x", "/c/Users/me", 4, "~/proj/x"},              // HOME collapse, 4 fields
		{"/c/Users/me/a/b/c/d", "/c/Users/me", 4, "~/a/…/d"},              // HOME collapse, 5 fields
		// Windows-native paths collapse on backslashes (spec: Path collapsing
		// supports Windows separators).
		{`C:\Users\demo\projects\deep\nested\coralline`, "", 4, `C:\Users\…\coralline`},
		{`C:\Users\demo\projects\deep\nested\coralline`, `C:\Users\demo`, 4, `~\projects\…\coralline`},
		{"/Users/demo/coralline", "", 4, "/Users/demo/coralline"}, // POSIX 4 fields → unchanged
		{"/Users/demo/projects/deep/nested/coralline", "", 4, "/Users/…/coralline"},
	}
	for _, c := range cases {
		if got := collapsePath(c.cwd, c.home, c.depth); got != c.want {
			t.Errorf("collapsePath(%q, %q, %d) = %q, want %q", c.cwd, c.home, c.depth, got, c.want)
		}
	}
}

// ctx token detail appears when VL_CTX_TOKENS != 0.
func TestSegCtxTokenDetail(t *testing.T) {
	c := conf.Defaults()
	c.Set("VL_CTX_TOKENS", "1")
	seg, ok := segCtx(c, Data{CtxPct: "50", TokIn: 1234, TokOut: 567, TokCR: 1234567, TokCW: 89})
	if !ok {
		t.Fatal("ctx should be visible")
	}
	if !strings.Contains(seg.Text, "↑1.2k ↓567 cr:1.2M cw:89 ") {
		t.Errorf("ctx token detail missing/incorrect: %q", seg.Text)
	}
}

func TestSegCtxNoTokenDetail(t *testing.T) {
	c := conf.Defaults()
	c.Set("VL_CTX_TOKENS", "0")
	seg, _ := segCtx(c, Data{CtxPct: "50", TokIn: 1234})
	if strings.Contains(seg.Text, "cr:") {
		t.Errorf("ctx should have no token detail when VL_CTX_TOKENS=0: %q", seg.Text)
	}
}

func TestGitDirtyMarks(t *testing.T) {
	c := conf.Defaults()
	st := gitstate.Parse("# branch.oid " + strings.Repeat("a", 40) + "\n# branch.head main\n1 M. x\n? y\n")
	seg, ok := segGit(c, Data{Git: st})
	if !ok {
		t.Fatal("git should be visible")
	}
	if !strings.Contains(seg.Text, "⎇ main+?") {
		t.Errorf("git marks wrong: %q", seg.Text)
	}
	if seg.BG != c.Get("VL_BG_GIT_DIRTY") {
		t.Errorf("dirty repo should use dirty bg")
	}
}

// A line whose segments are all hidden or unimplemented is omitted entirely.
func TestLayoutOmitsEmptyAndSkipsUnimplemented(t *testing.T) {
	c := conf.Defaults()
	c.Set("VL_SEGMENTS", "model")          // visible
	c.Set("VL_SEGMENTS2", "noseg1 noseg2") // unimplemented → skipped → line omitted
	c.Set("VL_SEGMENTS3", "git")           // hidden (no repo) → line omitted
	out := Statusline(c, Data{Model: "Claude Fable 5"})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly 1 line, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "Fable 5") {
		t.Errorf("line should render model: %q", lines[0])
	}
}

// VL_MAX_LINES caps the number of rendered lists.
func TestLayoutMaxLines(t *testing.T) {
	c := conf.Defaults()
	c.Set("VL_SEGMENTS", "model")
	c.Set("VL_SEGMENTS2", "effort")
	c.Set("VL_SEGMENTS3", "dir")
	c.Set("VL_MAX_LINES", "2")
	out := Statusline(c, Data{Model: "Claude X", Effort: "high", Cwd: "/a/b"})
	if n := strings.Count(out, "\n"); n != 2 {
		t.Errorf("VL_MAX_LINES=2 should yield 2 lines, got %d: %q", n, out)
	}
}

func TestSegCost(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segCost(c, Data{Cost: "1.2345"})
	if !ok {
		t.Fatal("cost should be visible")
	}
	if !strings.Contains(seg.Text, "$1.23") {
		t.Errorf("cost text wrong: %q", seg.Text)
	}
}

func TestSegCostZero(t *testing.T) {
	if _, ok := segCost(conf.Defaults(), Data{Cost: "0"}); ok {
		t.Error("cost should be hidden when zero")
	}
}

func TestSegCostEmpty(t *testing.T) {
	if _, ok := segCost(conf.Defaults(), Data{}); ok {
		t.Error("cost should be hidden when empty")
	}
}

func TestSegLines(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segLines(c, Data{LinesAdd: 42, LinesDel: 7})
	if !ok {
		t.Fatal("lines should be visible")
	}
	if !strings.Contains(seg.Text, "+42") || !strings.Contains(seg.Text, "-7") {
		t.Errorf("lines text wrong: %q", seg.Text)
	}
}

func TestSegLinesZero(t *testing.T) {
	if _, ok := segLines(conf.Defaults(), Data{}); ok {
		t.Error("lines should be hidden when both zero")
	}
}

func TestSegTokens(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segTokens(c, Data{CtxPct: "50", TokIn: 1000, TokOut: 500, TokCR: 2000, TokCW: 100})
	if !ok {
		t.Fatal("tokens should be visible")
	}
	if !strings.Contains(seg.Text, "↑1.0k") {
		t.Errorf("tokens text wrong: %q", seg.Text)
	}
}

func TestSegTokensHiddenNoCtx(t *testing.T) {
	if _, ok := segTokens(conf.Defaults(), Data{TokIn: 1000}); ok {
		t.Error("tokens should be hidden when ctx_pct empty")
	}
}

func TestSegStyle(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segStyle(c, Data{OutStyle: "concise"})
	if !ok {
		t.Fatal("style should be visible")
	}
	if !strings.Contains(seg.Text, "✎ concise") {
		t.Errorf("style text wrong: %q", seg.Text)
	}
}

func TestSegStyleDefault(t *testing.T) {
	if _, ok := segStyle(conf.Defaults(), Data{OutStyle: "default"}); ok {
		t.Error("style should be hidden for default")
	}
}

func TestSegDuration(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segDuration(c, Data{DurMs: 5025000})
	if !ok {
		t.Fatal("duration should be visible")
	}
	if !strings.Contains(seg.Text, "⧖ 1h23m") {
		t.Errorf("duration text wrong: %q", seg.Text)
	}
}

func TestSegDurationZero(t *testing.T) {
	if _, ok := segDuration(conf.Defaults(), Data{}); ok {
		t.Error("duration should be hidden when zero")
	}
}

func TestSegDurationSeconds(t *testing.T) {
	seg, ok := segDuration(conf.Defaults(), Data{DurMs: 45000})
	if !ok {
		t.Fatal("duration should be visible")
	}
	if !strings.Contains(seg.Text, "45s") {
		t.Errorf("duration text wrong: %q", seg.Text)
	}
}

func TestSegStash(t *testing.T) {
	c := conf.Defaults()
	st := gitstate.Parse("# branch.oid " + strings.Repeat("a", 40) + "\n# branch.head main\n")
	seg, ok := segStash(c, Data{Git: st, StashCount: 3})
	if !ok {
		t.Fatal("stash should be visible")
	}
	if !strings.Contains(seg.Text, "⚑ 3") {
		t.Errorf("stash text wrong: %q", seg.Text)
	}
}

func TestSegStashZero(t *testing.T) {
	st := gitstate.Parse("# branch.oid " + strings.Repeat("a", 40) + "\n# branch.head main\n")
	if _, ok := segStash(conf.Defaults(), Data{Git: st, StashCount: 0}); ok {
		t.Error("stash should be hidden when zero")
	}
}

func TestSegStashNoGit(t *testing.T) {
	if _, ok := segStash(conf.Defaults(), Data{StashCount: 5}); ok {
		t.Error("stash should be hidden outside git repo")
	}
}

func TestSegProject(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segProject(c, Data{GitRoot: "/home/user/my-project"})
	if !ok {
		t.Fatal("project should be visible")
	}
	if !strings.Contains(seg.Text, "⬢ my-project") {
		t.Errorf("project text wrong: %q", seg.Text)
	}
}

func TestSegProjectFallbackToDir(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segProject(c, Data{Cwd: "/tmp/work", SegScan: " model git "})
	if !ok {
		t.Fatal("project should fall back to dir")
	}
	if !strings.Contains(seg.Text, "/tmp/work") {
		t.Errorf("project fallback text wrong: %q", seg.Text)
	}
}

func TestSegProjectSuppressedWhenDirPresent(t *testing.T) {
	if _, ok := segProject(conf.Defaults(), Data{Cwd: "/tmp/work", SegScan: " dir git "}); ok {
		t.Error("project should be hidden when dir is already in segments and not in git repo")
	}
}

func TestSegNode(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segNode(c, Data{NodeVersion: "20.11.0"})
	if !ok {
		t.Fatal("node should be visible")
	}
	if !strings.Contains(seg.Text, "20.11.0") {
		t.Errorf("node text wrong: %q", seg.Text)
	}
}

func TestSegNodeEmpty(t *testing.T) {
	if _, ok := segNode(conf.Defaults(), Data{}); ok {
		t.Error("node should be hidden when empty")
	}
}

func TestSegPython(t *testing.T) {
	c := conf.Defaults()
	seg, ok := segPython(c, Data{PythonVersion: "myenv"})
	if !ok {
		t.Fatal("python should be visible")
	}
	if !strings.Contains(seg.Text, "myenv") {
		t.Errorf("python text wrong: %q", seg.Text)
	}
}

func TestSegPythonEmpty(t *testing.T) {
	if _, ok := segPython(conf.Defaults(), Data{}); ok {
		t.Error("python should be hidden when empty")
	}
}

func TestSegClockOff(t *testing.T) {
	c := conf.Defaults()
	c.Set("VL_CLOCK", "off")
	if _, ok := segClock(c, Data{Now: 1700000000}); ok {
		t.Error("clock should be hidden when off")
	}
}

func TestSegClock24h(t *testing.T) {
	c := conf.Defaults()
	c.Set("VL_CLOCK", "24h")
	c.Set("VL_CLOCK_SECONDS", "0")
	seg, ok := segClock(c, Data{Now: 1700000000})
	if !ok {
		t.Fatal("clock should be visible")
	}
	if !strings.Contains(seg.Text, "⊙") {
		t.Errorf("clock should have clock glyph: %q", seg.Text)
	}
}
