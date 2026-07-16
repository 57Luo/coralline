package render

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"coralline/internal/conf"
	"coralline/internal/inputjson"
)

// TestGoldenPill renders the fixed testdata input with the testdata config and
// compares byte-for-byte against golden_pill.txt, which was captured from the
// bash reference implementation (statusline.sh) on the same input and config.
// This is the cross-implementation oracle check for the pill layout and the
// deterministic segments (dir, git-hidden, model, effort, ctx, limit5h,
// limit7d-hidden, burn-warming).
func TestGoldenPill(t *testing.T) {
	confPath := filepath.Join("testdata", "coralline.conf")
	// $_VL_DIR is the renderer executable dir; in this flat testdata fixture the
	// theme lives directly under testdata/, so vlDir = testdata.
	c, err := conf.Load(confPath, "testdata")
	if err != nil {
		t.Fatalf("conf.Load: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join("testdata", "input.json"))
	if err != nil {
		t.Fatal(err)
	}
	in := inputjson.Parse(bytes.NewReader(raw))

	d := Data{
		Cwd:    in.Cwd,
		Home:   "", // no ~ collapse (deterministic; input path is not under HOME)
		Model:  in.Model,
		CtxPct: in.CtxPct,
		TokIn:  in.TokIn,
		TokOut: in.TokOut,
		TokCR:  in.TokCR,
		TokCW:  in.TokCW,
		Effort: in.Effort,
		Cost:   in.Cost,
		LinesAdd: in.LinesAdd,
		LinesDel: in.LinesDel,
		OutStyle: in.OutStyle,
		DurMs:    in.DurMs,
		// Git segment hidden: the input cwd is not a repository (matches how the
		// oracle was generated).
		Fh:        Limit{Pct: in.FhPct, Reset: in.FhRst},
		Wd:        Limit{Pct: in.WdPct, Reset: in.WdRst},
		Burn:      BurnResult{State: "warming"},
		BurnGuard: in.FhPct != "" || in.WdPct != "",
		Now:       1770000000, // well past resets_at (2020) → countdown "now", time-invariant
	}

	got := Statusline(c, d)

	want, err := os.ReadFile(filepath.Join("testdata", "golden_pill.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("golden mismatch:\n--- got ---\n%q\n--- want ---\n%q", got, string(want))
	}
}
