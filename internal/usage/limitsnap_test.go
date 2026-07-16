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

// Spec example: 7d window reset from 75% to 3% — same reset epoch, drop beyond
// the 5-point threshold purges the stale high-water.
func TestSampleResetPurgesSameEpochHighWater(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "limit-7d.tsv")
	now := int64(1784300000)
	mustMkdir(t, StorePath(file), "1784311200_075.000")

	if err := Sample(file, "3.0", "1784311200", now, RLMax7d); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(StorePath(file), "1784311200_075.000")); !os.IsNotExist(err) {
		t.Errorf("stale high-water entry should have been purged")
	}
	snap := Latest(file, now, RLMax7d)
	if !snap.OK || snap.Reset != 1784311200 || snap.Pct != "003.000" {
		t.Fatalf("Latest = %+v, want reset 1784311200 pct 003.000", snap)
	}
}

// Spec example: stale session reports 74.5 against a 75.0 high-water — a drop
// within the 5-point jitter threshold must not purge.
func TestSampleJitterWithinThresholdKeepsHighWater(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "limit-7d.tsv")
	now := int64(1784300000)
	mustMkdir(t, StorePath(file), "1784311200_075.000")

	if err := Sample(file, "74.5", "1784311200", now, RLMax7d); err != nil {
		t.Fatal(err)
	}
	snap := Latest(file, now, RLMax7d)
	if !snap.OK || snap.Pct != "075.000" {
		t.Fatalf("Latest = %+v, want high-water 075.000 preserved", snap)
	}
}

// Spec example: a new 7d window opening at a low percentage is not a reset —
// detection is scoped to entries sharing the sample's reset epoch.
func TestSampleDifferentEpochNoPurge(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "limit-7d.tsv")
	now := int64(1784300000)
	mustMkdir(t, StorePath(file), "1784311200_075.000")

	if err := Sample(file, "2.0", "1784916000", now, RLMax7d); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(StorePath(file), "1784311200_075.000")); err != nil {
		t.Errorf("old-window entry must not be deleted by reset detection")
	}
	if _, err := os.Stat(filepath.Join(StorePath(file), "1784916000_002.000")); err != nil {
		t.Errorf("new-window entry should have been recorded")
	}
}

// Reset purge failures stay silent: even when deleting the stale entry fails
// (here forced by making it non-empty; in production a concurrent render may
// have removed it first), Sample reports no error and still records the sample.
func TestSampleResetPurgeFailureStaysSilent(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "limit-7d.tsv")
	now := int64(1784300000)
	stale := "1784311200_075.000"
	mustMkdir(t, StorePath(file), stale)
	if err := os.WriteFile(filepath.Join(StorePath(file), stale, "child"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Sample(file, "3.0", "1784311200", now, RLMax7d); err != nil {
		t.Fatalf("Sample should ignore purge failures, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(StorePath(file), "1784311200_003.000")); err != nil {
		t.Errorf("incoming sample should still be recorded after purge failure")
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
