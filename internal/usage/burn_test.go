package usage

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBurn(t *testing.T, file, content string) {
	t.Helper()
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Active: >=2 integer-percent up-crossings inside the window spanning >= win/10.
func TestBurnActive(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "burn-5h.tsv")
	const now, win, trim = 10000, 600, 1500
	R := int64(11000)
	writeBurn(t, file, "9500\t10\t11000\n9600\t20\t11000\n9700\t30\t11000\n")

	b := BurnEta5h(file, now, win, trim, RLMax5h)
	if b.State != "active" {
		t.Fatalf("state = %q, want active", b.State)
	}
	// rate = (30-20)/(9700-9600) = 0.1 pct/s; eta = (100-30)/0.1 = 700s.
	if math.Abs(b.ETA-700) > 1 {
		t.Errorf("ETA = %v, want ~700", b.ETA)
	}
	if b.TTR != R-now {
		t.Errorf("TTR = %d, want %d", b.TTR, R-now)
	}
}

// Idle: an up-crossing exists in the current window but none within the recent
// time window (timestamps older than now-win).
func TestBurnIdle(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "burn-5h.tsv")
	const now, win, trim = 10000, 600, 1500
	// Timestamps 8000/8100 are older than cwin=9400; reset 11000 is the current
	// window.
	writeBurn(t, file, "8000\t10\t11000\n8100\t20\t11000\n")

	b := BurnEta5h(file, now, win, trim, RLMax5h)
	if b.State != "idle" {
		t.Fatalf("state = %q, want idle", b.State)
	}
	if !math.IsInf(b.ETA, 1) {
		t.Errorf("ETA = %v, want +Inf", b.ETA)
	}
}

// Warming: no up-crossings at all (single sample).
func TestBurnWarming(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "burn-5h.tsv")
	writeBurn(t, file, "9500\t50\t11000\n")
	b := BurnEta5h(file, 10000, 600, 1500, RLMax5h)
	if b.State != "warming" {
		t.Fatalf("state = %q, want warming", b.State)
	}
}

func TestBurnWarmingEmptyFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "burn-5h.tsv")
	writeBurn(t, file, "")
	b := BurnEta5h(file, 10000, 600, 1500, RLMax5h)
	if b.State != "warming" {
		t.Errorf("state = %q, want warming for empty file", b.State)
	}
}

func TestBurnMissingFileWarming(t *testing.T) {
	b := BurnEta5h(filepath.Join(t.TempDir(), "nope.tsv"), 10000, 600, 1500, RLMax5h)
	if b.State != "warming" {
		t.Errorf("state = %q, want warming for missing file", b.State)
	}
}

// A sentinel row (reset far beyond now+maxAhead) is dropped and the file is
// rewritten without it (self-healing).
func TestBurnSentinelDroppedAndFileHealed(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "burn-5h.tsv")
	const now, win, trim = 10000, 600, 1500
	// One legitimate row plus a 2030-style sentinel reset far in the future.
	writeBurn(t, file, "9500\t50\t11000\n9600\t60\t9999999999\n")

	BurnEta5h(file, now, win, trim, RLMax5h)

	data, _ := os.ReadFile(file)
	if strings.Contains(string(data), "9999999999") {
		t.Errorf("sentinel row should be healed out of the file, got:\n%s", data)
	}
	if !strings.Contains(string(data), "9500\t50\t11000") {
		t.Errorf("legitimate row should survive, got:\n%s", data)
	}
}

// Trim keeps at most `trim` distinct-epoch rows (physical-row overflow triggers
// the atomic rewrite).
func TestBurnTrim(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "burn-5h.tsv")
	// 5 distinct-epoch rows, trim=3 → file rewritten to last 3.
	writeBurn(t, file, "100\t1\t11000\n200\t2\t11000\n300\t3\t11000\n400\t4\t11000\n500\t5\t11000\n")
	BurnEta5h(file, 10000, 600, 3, RLMax5h)

	data, _ := os.ReadFile(file)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 rows after trim, got %d:\n%s", len(lines), data)
	}
	// Must keep the most recent 3 (300,400,500), drop the oldest.
	if !strings.HasPrefix(lines[0], "300\t") || !strings.HasPrefix(lines[2], "500\t") {
		t.Errorf("trim should keep the most recent rows, got:\n%s", data)
	}
}
