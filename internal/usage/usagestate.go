package usage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// SerializeUsageState renders the usage-state.json line exactly as the bash
// printf template does: fixed field order, raw (unescaped) substitution of the
// model and percentages, trailing newline. Empty percentages become 0
// (bash ${fh_pct:-0} / ${wd_pct:-0}).
func SerializeUsageState(now int64, model, fhPct, fhRst, wdPct, wdRst string) string {
	if fhPct == "" {
		fhPct = "0"
	}
	if wdPct == "" {
		wdPct = "0"
	}
	return fmt.Sprintf(
		`{"source":"coralline","updated_at":%d,"model":"%s","five_hour":{"used_percentage":%s,"resets_at":"%s"},"seven_day":{"used_percentage":%s,"resets_at":"%s"}}`+"\n",
		now, model, fhPct, fhRst, wdPct, wdRst)
}

// WriteUsageState writes the snapshot to path atomically (temp file + rename)
// and sweeps temp files orphaned by dead sessions. Callers gate this on
// VL_USAGE_STATE=1 and a non-empty five-hour percentage.
func WriteUsageState(path string, now int64, model, fhPct, fhRst, wdPct, wdRst string) error {
	content := SerializeUsageState(now, model, fhPct, fhRst, wdPct, wdRst)
	tmp := path + ".tmp." + strconv.Itoa(os.Getpid())

	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	// Sweep temps orphaned by dead sessions (all but our own, already renamed).
	if matches, err := filepath.Glob(path + ".tmp.*"); err == nil {
		for _, m := range matches {
			if m != tmp {
				_ = os.Remove(m)
			}
		}
	}
	return nil
}
