package usage

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"coralline/internal/epoch"
)

// BurnState is the 5h burn projection (statusline.sh burn_eta_5h output).
type BurnState struct {
	State string  // "active" | "idle" | "warming"
	ETA   float64 // seconds until empty, or +Inf
	Rate  float64 // pct per second
	TTR   int64   // seconds until the window resets
}

// Append records one 5h burn sample (statusline.sh burn_sample): a
// `<now>\t<pct>\t<reset-epoch>` row. An empty percentage, an unparseable reset,
// or a reset beyond now+maxAhead is ignored.
func Append(file, pctStr, resetStr string, now, maxAhead int64) error {
	if pctStr == "" {
		return nil
	}
	ep, ok := epoch.ToEpoch(resetStr)
	if !ok {
		return nil
	}
	if ep > now+maxAhead {
		return nil
	}
	if dir := filepath.Dir(file); dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			_ = os.MkdirAll(dir, 0o755)
		}
	}
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%d\t%s\t%d\n", now, pctStr, ep)
	return err
}

// BurnEta5h is a faithful port of the burn_eta_5h awk reference. It restricts
// samples to the current window (greatest reset), counts integer-percent
// up-crossings within the recent time window, and yields active/idle/warming.
// As a side effect it drops poisoned sentinel rows and trims the file to `trim`
// physical rows via an atomic rewrite.
func BurnEta5h(file string, now, win, trim, maxAhead int64) BurnState {
	data, err := os.ReadFile(file)
	if err != nil {
		return BurnState{State: "warming", ETA: math.Inf(1)}
	}

	lines := strings.Split(string(data), "\n")
	// A trailing newline yields a final empty element; drop it so NR matches awk.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	nr := int64(len(lines))

	var ord []int64
	seen := map[int64]bool{}
	pct := map[int64]float64{}
	rst := map[int64]int64{}
	dropped := false

	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 || fields[1] == "" {
			continue // awk `$2 != ""`
		}
		e := int64(parseNum(fields[0]))
		var r int64
		if len(fields) >= 3 {
			r = int64(parseNum(fields[2]))
		}
		if r > now+maxAhead {
			dropped = true
			continue
		}
		if !seen[e] {
			ord = append(ord, e)
			seen[e] = true
		}
		pct[e] = parseNum(fields[1])
		rst[e] = r
	}

	n := int64(len(ord))
	result := BurnState{State: "warming", ETA: math.Inf(1)}

	if n == 0 {
		result = BurnState{State: "warming", ETA: math.Inf(1)}
	} else {
		// Current window = the greatest reset; keep its samples in file order.
		var cur int64
		for _, e := range ord {
			if rst[e] > cur {
				cur = rst[e]
			}
		}
		var cord []int64
		for _, e := range ord {
			if rst[e] == cur {
				cord = append(cord, e)
			}
		}
		m := len(cord)
		le := cord[m-1]
		lp := pct[le]
		ttr := cur - now
		if ttr < 0 {
			ttr = 0
		}
		cwin := now - win
		minspan := win / 10

		var fcT, lcT int64
		fcP, lcP := -1, -1
		ncross := 0
		anycross := false
		for i := 1; i < m; i++ {
			a := int(pct[cord[i-1]])
			b := int(pct[cord[i]])
			if b > a {
				anycross = true
				ct := cord[i]
				if ct >= cwin && ct <= now {
					if fcP < 0 {
						fcT = ct
						fcP = b
					}
					lcT = ct
					lcP = b
					ncross++
				}
			}
		}

		switch {
		case ncross >= 2 && lcT > fcT && lcP > fcP && (lcT-fcT) >= minspan:
			rate := float64(lcP-fcP) / float64(lcT-fcT)
			eta := (100 - lp) / rate
			if eta < 0 {
				eta = 0
			}
			result = BurnState{State: "active", ETA: math.Round(eta), Rate: rate, TTR: ttr}
		case anycross && ncross == 0:
			result = BurnState{State: "idle", ETA: math.Inf(1), TTR: ttr}
		default:
			result = BurnState{State: "warming", ETA: math.Inf(1), TTR: ttr}
		}

		// Trim on physical rows (or when a sentinel was dropped), rewriting the
		// deduped last-`trim` epochs atomically.
		if nr > trim || dropped {
			lo := n - trim + 1
			if lo < 1 {
				lo = 1
			}
			var b strings.Builder
			for i := lo; i <= n; i++ {
				e := ord[i-1]
				fmt.Fprintf(&b, "%d\t%s\t%d\n", e, formatPct(pct[e]), rst[e])
			}
			rewriteAtomic(file, b.String())
		}
	}

	return result
}

// Estimate is the combined burn projection the burn segment renders
// (statusline.sh burn_estimate output). State is "active", "idle", or "warming";
// Label is "5h"/"7d" when active.
type Estimate struct {
	State string
	Label string
	ETA   int64
	TTR   int64
}

// BurnEstimate combines the 5h and 7d projections exactly as statusline.sh
// burn_estimate does: the binding limit is whichever runs empty sooner. When
// limitSync is on, the 7d projection uses the synced high-water (so burn and the
// limit7d segment cannot contradict each other on a stale local snapshot).
func BurnEstimate(burnFile string, now, burnWin, trim int64, limitSync bool, rl7dFile, wdPct, wdRst string) Estimate {
	b5 := BurnEta5h(burnFile, now, burnWin, trim, RLMax5h)

	var b7 BurnState
	if limitSync {
		if snap := Latest(rl7dFile, now, RLMax7d); snap.OK {
			b7 = BurnEta7d(snap.Pct, strconv.FormatInt(snap.Reset, 10), now)
		} else {
			b7 = BurnEta7d(wdPct, wdRst, now)
		}
	} else {
		b7 = BurnEta7d(wdPct, wdRst, now)
	}

	f5 := !math.IsInf(b5.ETA, 1)
	f7 := !math.IsInf(b7.ETA, 1)
	switch {
	case f5 && (!f7 || b5.ETA <= b7.ETA):
		return Estimate{State: "active", Label: "5h", ETA: int64(b5.ETA), TTR: b5.TTR}
	case f7:
		return Estimate{State: "active", Label: "7d", ETA: int64(b7.ETA), TTR: b7.TTR}
	default:
		st := "warming"
		if b5.State == "idle" {
			st = "idle"
		}
		return Estimate{State: st}
	}
}

// BurnEta7d is the stateless 7d projection (statusline.sh burn_eta_7d).
func BurnEta7d(pctStr, resetStr string, now int64) BurnState {
	res := BurnState{ETA: math.Inf(1)}
	if pctStr == "" {
		return res
	}
	ep, ok := epoch.ToEpoch(resetStr)
	if !ok {
		return res
	}
	p := parseNum(pctStr)
	ttr := ep - now
	if ttr < 0 {
		ttr = 0
	}
	res.TTR = ttr
	ws := ep - 7*86400
	el := now - ws
	if p <= 0 || el <= 0 {
		return res
	}
	rate := p / float64(el)
	eta := (100 - p) / rate
	if eta < 0 {
		eta = 0
	}
	res.ETA = math.Round(eta)
	res.Rate = rate
	return res
}

// parseNum mirrors awk's `$x + 0`: leading-numeric coercion, 0 on failure.
func parseNum(s string) float64 {
	s = strings.TrimSpace(s)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return 0
}

// formatPct matches awk's default number→string conversion (CONVFMT="%.6g").
func formatPct(f float64) string {
	return strconv.FormatFloat(f, 'g', 6, 64)
}

// rewriteAtomic writes content to file via a temp file + rename, then sweeps
// temps orphaned by dead sessions.
func rewriteAtomic(file, content string) {
	tmp := file + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		_ = os.Remove(tmp)
		return
	}
	if err := os.Rename(tmp, file); err != nil {
		_ = os.Remove(tmp)
		return
	}
	if matches, err := filepath.Glob(file + ".*.tmp"); err == nil {
		for _, m := range matches {
			if m != tmp {
				_ = os.Remove(m)
			}
		}
	}
}
