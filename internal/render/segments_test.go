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
	c.Set("VL_SEGMENTS", "model")       // visible
	c.Set("VL_SEGMENTS2", "cost clock") // unimplemented → skipped → line omitted
	c.Set("VL_SEGMENTS3", "git")        // hidden (no repo) → line omitted
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
