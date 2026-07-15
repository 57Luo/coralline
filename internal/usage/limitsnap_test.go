package usage

import (
	"os"
	"path/filepath"
	"testing"
)

// Example from the spec: reset 1770000000, pct 41.25 → dir "1770000000_041.250".
func TestSnapshotNameEncoding(t *testing.T) {
	if got := snapshotName(1770000000, 41.25); got != "1770000000_041.250" {
		t.Errorf("snapshotName = %q, want 1770000000_041.250", got)
	}
}

func TestSampleAndLatestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "limit-5h.tsv")
	// now=1769000000 so reset 1770000000 is within the 5h ceiling? No: reset is
	// ~1.16M s ahead, far beyond RL_MAX_5H — use a now close to reset.
	now := int64(1769999000)
	if err := Sample(file, "41.25", "1770000000", now, RLMax5h); err != nil {
		t.Fatal(err)
	}
	store := StorePath(file)
	entries, _ := os.ReadDir(store)
	if len(entries) != 1 || entries[0].Name() != "1770000000_041.250" {
		t.Fatalf("store entries = %v, want [1770000000_041.250]", names(entries))
	}

	snap := Latest(file, now, RLMax5h)
	if !snap.OK {
		t.Fatal("Latest returned not-ok")
	}
	if snap.Reset != 1770000000 {
		t.Errorf("Reset = %d, want 1770000000", snap.Reset)
	}
	if snap.Pct != "041.250" {
		t.Errorf("Pct = %q, want 041.250", snap.Pct)
	}
}

func TestLatestSelectsMaxAndGCsRest(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "limit-7d.tsv")
	now := int64(1769000000)
	// Three entries in the same window family; the greatest reset wins.
	mustMkdir(t, StorePath(file), "1769100000_010.000")
	mustMkdir(t, StorePath(file), "1769200000_055.500")
	mustMkdir(t, StorePath(file), "1769150000_030.000")

	snap := Latest(file, now, RLMax7d)
	if !snap.OK || snap.Reset != 1769200000 || snap.Pct != "055.500" {
		t.Fatalf("Latest = %+v, want reset 1769200000 pct 055.500", snap)
	}
	entries, _ := os.ReadDir(StorePath(file))
	if len(entries) != 1 || entries[0].Name() != "1769200000_055.500" {
		t.Errorf("store should have single surviving entry, got %v", names(entries))
	}
}

func TestLatestPrunesPoisonedSentinel(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "limit-5h.tsv")
	now := int64(1769000000)
	// A legitimate current entry plus a 2030 sentinel far beyond the 5h ceiling.
	mustMkdir(t, StorePath(file), "1769001000_042.000")
	sentinel := "1900000000_099.000" // ~2030, far past now+RLMax5h
	mustMkdir(t, StorePath(file), sentinel)

	snap := Latest(file, now, RLMax5h)
	if !snap.OK || snap.Reset != 1769001000 {
		t.Fatalf("Latest = %+v, want the legitimate entry, sentinel pruned", snap)
	}
	if _, err := os.Stat(filepath.Join(StorePath(file), sentinel)); !os.IsNotExist(err) {
		t.Errorf("sentinel entry should have been pruned")
	}
}

func TestSampleRejectsFarFuture(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "limit-5h.tsv")
	now := int64(1769000000)
	// Reset far beyond now+RLMax5h must not be recorded.
	if err := Sample(file, "50", "1900000000", now, RLMax5h); err != nil {
		t.Fatal(err)
	}
	store := StorePath(file)
	if entries, _ := os.ReadDir(store); len(entries) != 0 {
		t.Errorf("far-future reset should not be recorded, got %v", names(entries))
	}
}

func names(e []os.DirEntry) []string {
	var s []string
	for _, x := range e {
		s = append(s, x.Name())
	}
	return s
}

func mustMkdir(t *testing.T, store, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(store, name), 0o755); err != nil {
		t.Fatal(err)
	}
}
