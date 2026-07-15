package usage

import (
	"os"
	"path/filepath"
	"testing"
)

// Example from the spec (usage-state.json byte-compatible export).
func TestSerializeUsageStateExample(t *testing.T) {
	got := SerializeUsageState(1770000000, "Fable 5", "41", "2026-07-14T10:00:00Z", "79", "2026-07-18T00:00:00Z")
	want := `{"source":"coralline","updated_at":1770000000,"model":"Fable 5","five_hour":{"used_percentage":41,"resets_at":"2026-07-14T10:00:00Z"},"seven_day":{"used_percentage":79,"resets_at":"2026-07-18T00:00:00Z"}}` + "\n"
	if got != want {
		t.Errorf("serialize mismatch:\n got %q\nwant %q", got, want)
	}
}

// Empty seven-day percentage serializes as 0 (bash ${wd_pct:-0}).
func TestSerializeUsageStateEmptyWd(t *testing.T) {
	got := SerializeUsageState(100, "M", "5", "R1", "", "")
	want := `{"source":"coralline","updated_at":100,"model":"M","five_hour":{"used_percentage":5,"resets_at":"R1"},"seven_day":{"used_percentage":0,"resets_at":""}}` + "\n"
	if got != want {
		t.Errorf("serialize mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestWriteUsageStateAtomicNoTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage-state.json")
	if err := WriteUsageState(path, 1770000000, "Fable 5", "41", "2026-07-14T10:00:00Z", "79", "2026-07-18T00:00:00Z"); err != nil {
		t.Fatalf("WriteUsageState: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := SerializeUsageState(1770000000, "Fable 5", "41", "2026-07-14T10:00:00Z", "79", "2026-07-18T00:00:00Z")
	if string(data) != want {
		t.Errorf("file content = %q, want %q", string(data), want)
	}
	// No temp file must remain.
	temps, _ := filepath.Glob(path + ".tmp.*")
	if len(temps) != 0 {
		t.Errorf("orphan temp files remain: %v", temps)
	}
}

// A stale temp file from a dead session is swept.
func TestWriteUsageStateSweepsOrphanTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage-state.json")
	orphan := path + ".tmp.99999"
	if err := os.WriteFile(orphan, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteUsageState(path, 1, "M", "1", "", "0", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Errorf("orphan temp %s should have been swept", orphan)
	}
}
