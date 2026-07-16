// Command coralline renders the Claude Code statusline from stdin session JSON.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"coralline/internal/conf"
	"coralline/internal/gitstate"
	"coralline/internal/inputjson"
	"coralline/internal/render"
	"coralline/internal/runtime"
	"coralline/internal/usage"
)

// watchdogTimeout is the hard in-process deadline. It matches the bash
// implementation's `read -t 5` calibration (see design: watchdog decision).
const watchdogTimeout = 5 * time.Second

func main() {
	// Arm the hard deadline first thing: whatever the main logic blocks on
	// (stdin read, file I/O, child-process wait), the process still dies with
	// exit code 1 and no stdout output.
	time.AfterFunc(watchdogTimeout, func() { os.Exit(1) })

	in := inputjson.Parse(os.Stdin)
	out := renderStatusline(in, time.Now().Unix())
	// Any error is already degraded into the output; never surface error text.
	_, _ = os.Stdout.WriteString(out)
	os.Exit(0)
}

// renderStatusline runs the full pipeline: load config, run git, update the data
// files, and render the pill layout. Every failure degrades (hidden segment /
// built-in default); nothing is written to stdout but the statusline itself.
func renderStatusline(in inputjson.Input, now int64) string {
	// $_VL_DIR is the renderer executable's directory (bash _VL_DIR="${0%/*}").
	// _VL_CONFIG_DIR is its parent (bash ${_VL_DIR%/*}).
	vlDir := executableDir()
	configDir := filepath.Dir(vlDir)

	c, _ := conf.Load(conf.ResolvePath(configDir), vlDir)

	// Data-file paths: bash _VL_DIR/_VL_CONFIG_DIR semantics, with the same env
	// overrides the bash version honors.
	burnFile := envOr("CORALLINE_BURN_FILE", filepath.Join(vlDir, "burn-5h.tsv"))
	rl5hFile := envOr("CORALLINE_RL5H_FILE", filepath.Join(vlDir, "limit-5h.tsv"))
	rl7dFile := envOr("CORALLINE_RL7D_FILE", filepath.Join(vlDir, "limit-7d.tsv"))

	limitSync := c.Get("VL_LIMIT_SYNC") == "1"
	burnWin := c.GetInt("CORALLINE_BURN_WINDOW", 600)
	trim := c.GetInt("BURN_TRIM", 1500)

	// Segment scan across all three lists (space-padded, mirrors bash _SEG_SCAN).
	scan := " " + c.Get("VL_SEGMENTS") + " " + c.Get("VL_SEGMENTS2") + " " + c.Get("VL_SEGMENTS3") + " "
	has := func(name string) bool { return strings.Contains(scan, " "+name+" ") }

	// usage-state snapshot for external hooks (whitelisted fields only).
	if c.Get("VL_USAGE_STATE") == "1" && in.FhPct != "" {
		usFile := envOr("VL_USAGE_STATE_FILE", filepath.Join(configDir, "usage-state.json"))
		_ = usage.WriteUsageState(usFile, now, in.Model, in.FhPct, in.FhRst, in.WdPct, in.WdRst)
	}

	// Git state collection, only when a git segment is present. Run spawns up
	// to three git subprocesses (status, rev-parse --show-toplevel, rev-list
	// --count) sharing one timeout context that bounds them together.
	var git gitstate.State
	if has("git") || has("stash") || has("project") {
		git = gitstate.Run(context.Background(), in.Cwd)
	}

	// Record this render into the cross-session stores, unless this is a
	// preview/verification render (CORALLINE_NO_SAMPLE=1) — sample-input.json
	// carries a far-future sentinel that must not poison the real stores.
	if os.Getenv("CORALLINE_NO_SAMPLE") != "1" {
		if has("burn") {
			_ = usage.Append(burnFile, in.FhPct, in.FhRst, now, usage.RLMax5h)
		}
		if limitSync {
			if has("limit5h") {
				_ = usage.Sample(rl5hFile, in.FhPct, in.FhRst, now, usage.RLMax5h)
			}
			// burn also consumes the synced 7d, so sample it whenever burn shows.
			if has("limit7d") || has("burn") {
				_ = usage.Sample(rl7dFile, in.WdPct, in.WdRst, now, usage.RLMax7d)
			}
		}
	}

	// Resolve the limit gauges. With limit-sync, show the freshest cross-session
	// high-water for the current window, falling back to this session's value.
	fh := render.Limit{Pct: in.FhPct, Reset: in.FhRst}
	wd := render.Limit{Pct: in.WdPct, Reset: in.WdRst}
	if limitSync {
		if has("limit5h") {
			if snap := usage.Latest(rl5hFile, now, usage.RLMax5h); snap.OK {
				fh = render.Limit{Pct: snap.Pct, Reset: strconv.FormatInt(snap.Reset, 10)}
			}
		}
		if has("limit7d") {
			if snap := usage.Latest(rl7dFile, now, usage.RLMax7d); snap.OK {
				wd = render.Limit{Pct: snap.Pct, Reset: strconv.FormatInt(snap.Reset, 10)}
			}
		}
	}

	// Burn projection.
	var burn usage.Estimate
	if has("burn") {
		burn = usage.BurnEstimate(burnFile, now, int64(burnWin), int64(trim), limitSync, rl7dFile, in.WdPct, in.WdRst)
	}

	// Runtime detection for node/python segments.
	probe := c.Get("VL_RUNTIME_PROBE") == "1"
	var nodeVer, pyVer string
	if has("node") {
		nodeVer = runtime.DetectNode(in.Cwd, probe)
	}
	if has("python") {
		pyVer = runtime.DetectPython(in.Cwd, probe)
	}

	d := render.Data{
		Cwd:    in.Cwd,
		Home:   homeDir(),
		Model:  in.Model,
		CtxPct: in.CtxPct,
		TokIn:  in.TokIn,
		TokOut: in.TokOut,
		TokCR:  in.TokCR,
		TokCW:  in.TokCW,
		Effort: in.Effort,
		Git:    git,
		Fh:     fh,
		Wd:     wd,
		Burn: render.BurnResult{
			State: burn.State,
			Label: burn.Label,
			ETA:   burn.ETA,
			TTR:   burn.TTR,
		},
		BurnGuard: in.FhPct != "" || in.WdPct != "",
		Now:       now,

		Cost:          in.Cost,
		LinesAdd:      in.LinesAdd,
		LinesDel:      in.LinesDel,
		OutStyle:      in.OutStyle,
		DurMs:         in.DurMs,
		StashCount:    git.StashCount,
		GitRoot:       git.Root,
		NodeVersion:   nodeVer,
		PythonVersion: pyVer,
		SegScan:       scan,
	}
	return render.Statusline(c, d)
}

// executableDir returns the directory containing the running executable, falling
// back to the argv[0] directory (matching bash's ${0%/*}).
func executableDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return filepath.Dir(os.Args[0])
}

// homeDir returns $HOME (as the bash version uses) with a fallback.
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
