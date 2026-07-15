package usage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"coralline/internal/epoch"
)

// Per-window ceilings for the sentinel guard (statusline.sh RL_MAX_5H/RL_MAX_7D):
// a reset further out than its window can possibly be is corrupt and must never
// become the high-water.
const (
	RLMax5h = 6 * 3600  // 5h window resets within ~5h (+1h skew margin)
	RLMax7d = 8 * 86400 // 7d window resets within 7d (+1d skew margin)
)

// StorePath maps a limit .tsv file to its directory-set store (bash rl_dir):
// `<file without .tsv>.d`.
func StorePath(file string) string {
	return strings.TrimSuffix(file, ".tsv") + ".d"
}

// snapshotName encodes an entry name: fixed-width so lexical order equals numeric
// order (reset dominates, pct tie-breaks).
func snapshotName(resetEpoch int64, pct float64) string {
	return fmt.Sprintf("%010d_%07.3f", resetEpoch, pct)
}

// Snapshot is the resolved high-water reading.
type Snapshot struct {
	Pct   string // percentage as stored (fixed-width string, e.g. "041.250")
	Reset int64  // reset epoch
	OK    bool
}

// Sample records one high-water snapshot as an atomically-created empty
// directory. An empty percentage, an unparseable reset, or a reset beyond
// now+maxAhead (poisoned sentinel) is silently ignored. now/maxAhead of 0
// disables the far-future guard (mirrors bash's unit-call path).
func Sample(file, pctStr, resetStr string, now, maxAhead int64) error {
	if pctStr == "" {
		return nil
	}
	ep, ok := epoch.ToEpoch(resetStr)
	if !ok {
		return nil
	}
	if now != 0 && maxAhead != 0 && ep > now+maxAhead {
		return nil
	}
	pct, err := strconv.ParseFloat(pctStr, 64)
	if err != nil {
		return nil
	}
	store := StorePath(file)
	if _, err := os.Stat(store); os.IsNotExist(err) {
		if err := os.MkdirAll(store, 0o755); err != nil {
			return err
		}
		// One-shot migration: drop any pre-dir-set flat file + tmps.
		_ = os.Remove(file)
		if tmps, e := filepath.Glob(file + ".*.tmp"); e == nil {
			for _, t := range tmps {
				_ = os.Remove(t)
			}
		}
	}
	// Mkdir is atomic; a concurrent identical add is idempotent (already-exists
	// is not an error worth surfacing).
	_ = os.Mkdir(filepath.Join(store, snapshotName(ep, pct)), 0o755)
	return nil
}

// Latest reads the store: it selects the entry with the greatest reset epoch,
// prunes entries beyond now+win (poisoned sentinels), removes all non-max
// entries, and returns the surviving high-water. now/win of 0 disables sentinel
// pruning. A concurrent higher add is never lost: post-snapshot entries are not
// deletion candidates because deletion targets only entries below the chosen max
// from a single listing.
func Latest(file string, now, win int64) Snapshot {
	store := StorePath(file)
	entries, err := os.ReadDir(store)
	if err != nil {
		return Snapshot{}
	}

	var cut int64
	hasCut := now != 0 && win != 0
	if hasCut {
		cut = now + win
	}

	var kept []string
	for _, e := range entries {
		name := e.Name()
		if name == "" || name[0] < '0' || name[0] > '9' {
			continue // ignore non-snapshot entries (bash grep '^[0-9]')
		}
		reset, ok := parseReset(name)
		if !ok {
			continue
		}
		if hasCut && reset > cut {
			_ = os.Remove(filepath.Join(store, name)) // prune poisoned sentinel
			continue
		}
		kept = append(kept, name)
	}
	if len(kept) == 0 {
		return Snapshot{}
	}
	sort.Strings(kept) // fixed-width names → lexical == numeric order
	hi := kept[len(kept)-1]

	for _, name := range kept {
		if name != hi {
			_ = os.Remove(filepath.Join(store, name))
		}
	}

	reset, _ := parseReset(hi)
	pct := hi[strings.IndexByte(hi, '_')+1:]
	return Snapshot{Pct: pct, Reset: reset, OK: true}
}

// parseReset extracts the reset epoch (before the '_') from an entry name,
// base-10 so leading zeros are not read as octal.
func parseReset(name string) (int64, bool) {
	i := strings.IndexByte(name, '_')
	if i < 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(name[:i], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
