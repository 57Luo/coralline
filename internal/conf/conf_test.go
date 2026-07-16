package conf

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a helper to lay down a config/theme file under dir.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// GetInt parses a config value as int, falling back to def when the value is
// empty or not a number (same semantics as the former per-package confInt).
func TestGetInt(t *testing.T) {
	c := &Config{values: map[string]string{
		"VL_BAR_WIDTH":  "7",
		"VL_PADDED":     " 3 ",
		"VL_NOT_NUMBER": "wide",
	}}
	if got := c.GetInt("VL_BAR_WIDTH", 5); got != 7 {
		t.Errorf("GetInt valid = %d, want 7", got)
	}
	if got := c.GetInt("VL_PADDED", 5); got != 3 {
		t.Errorf("GetInt padded = %d, want 3", got)
	}
	if got := c.GetInt("VL_UNSET", 5); got != 5 {
		t.Errorf("GetInt empty = %d, want default 5", got)
	}
	if got := c.GetInt("VL_NOT_NUMBER", 5); got != 5 {
		t.Errorf("GetInt non-numeric = %d, want default 5", got)
	}
}

// Scenario: User config with theme source is honored, using the REAL deployment
// two-level layout: the config lives at <root>/coralline.conf while the renderer
// executable dir ($_VL_DIR) is <root>/coralline, one level below — so the theme
// resolves under <root>/coralline/themes, NOT next to the config file. This is
// the layout the earlier "conf-dir" semantics silently broke.
// Also covers double/single/bare quoting and precedence (later overwrites earlier).
func TestThemeSourceAndQuoting(t *testing.T) {
	root := t.TempDir()
	vlDir := filepath.Join(root, "coralline") // renderer executable directory
	writeFile(t, filepath.Join(vlDir, "themes", "test-theme.conf"), `
# a theme file
VL_BG_DIR="137,180,250"
VL_FG_TEXT="30,30,46"
VL_STYLE="from-theme"
`)
	confPath := filepath.Join(root, "coralline.conf") // one level ABOVE vlDir
	writeFile(t, confPath, `
# user config
VL_STYLE="lean"
. "$_VL_DIR/themes/test-theme.conf"
VL_STYLE="pill"
VL_SEGMENTS='ctx git'
VL_MAX_LINES=3
`)

	c, err := Load(confPath, vlDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Theme colors are applied (double-quoted triple).
	if got := c.Get("VL_BG_DIR"); got != "137,180,250" {
		t.Errorf("VL_BG_DIR = %q, want 137,180,250", got)
	}
	if got := c.Get("VL_FG_TEXT"); got != "30,30,46" {
		t.Errorf("VL_FG_TEXT = %q, want 30,30,46", got)
	}
	// VL_STYLE: user sets lean, theme sets from-theme, user re-sets pill after
	// source; last assignment wins → pill.
	if got := c.Get("VL_STYLE"); got != "pill" {
		t.Errorf("VL_STYLE = %q, want pill (last assignment wins)", got)
	}
	// Single-quoted value.
	if got := c.Get("VL_SEGMENTS"); got != "ctx git" {
		t.Errorf("VL_SEGMENTS = %q, want 'ctx git'", got)
	}
	// Bare value.
	if got := c.Get("VL_MAX_LINES"); got != "3" {
		t.Errorf("VL_MAX_LINES = %q, want 3", got)
	}
}

// Scenario: Unsupported syntax lines are ignored; rendering proceeds using the
// remaining assignments.
func TestUnsupportedLinesIgnored(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "coralline.conf")
	writeFile(t, confPath, `
VL_STYLE="pill"
if [ "$VL_ASCII" = "1" ]; then
  VL_CAP_L=""
fi
[ -f "$VL_CONF" ] && . "$VL_CONF"
export GIT_OPTIONAL_LOCKS=0
VL_WARN_PCT=50
`)
	c, err := Load(confPath, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := c.Get("VL_STYLE"); got != "pill" {
		t.Errorf("VL_STYLE = %q, want pill", got)
	}
	if got := c.Get("VL_WARN_PCT"); got != "50" {
		t.Errorf("VL_WARN_PCT = %q, want 50", got)
	}
	// The indented VL_CAP_L="" inside the if-block is a supported-looking
	// assignment line; bash would only run it conditionally, but our subset
	// parser treats any VAR=value line as an assignment. That is acceptable per
	// design (generated configs never contain conditional assignments); assert
	// the unsupported control-flow lines did not corrupt parsing of real keys.
}

// Defaults are present before any file load and overridden by the file.
func TestDefaultsAndOverride(t *testing.T) {
	c := Defaults()
	if got := c.Get("VL_STYLE"); got != "pill" {
		t.Errorf("default VL_STYLE = %q, want pill", got)
	}
	if got := c.Get("VL_HOT_PCT"); got != "75" {
		t.Errorf("default VL_HOT_PCT = %q, want 75", got)
	}

	dir := t.TempDir()
	confPath := filepath.Join(dir, "coralline.conf")
	writeFile(t, confPath, "VL_HOT_PCT=80\n")
	c2, err := Load(confPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := c2.Get("VL_HOT_PCT"); got != "80" {
		t.Errorf("VL_HOT_PCT = %q, want 80 (file overrides default)", got)
	}
	// A key not set by the file keeps its default.
	if got := c2.Get("VL_WARN_PCT"); got != "50" {
		t.Errorf("VL_WARN_PCT = %q, want 50 (default retained)", got)
	}
}

// VL_CONFIG environment variable overrides the path (spec), with the bash
// implementation's CORALLINE_CONFIG honored as a fallback; otherwise the default
// is <configDir>/coralline.conf (bash _VL_CONFIG_DIR semantics).
func TestResolvePath(t *testing.T) {
	dir := t.TempDir()
	vlPath := filepath.Join(dir, "vl.conf")
	coPath := filepath.Join(dir, "co.conf")
	configDir := filepath.Join(dir, "cfgdir")

	t.Setenv("VL_CONFIG", vlPath)
	t.Setenv("CORALLINE_CONFIG", coPath)
	if got := ResolvePath(configDir); got != vlPath {
		t.Errorf("ResolvePath with VL_CONFIG = %q, want %q", got, vlPath)
	}

	os.Unsetenv("VL_CONFIG")
	if got := ResolvePath(configDir); got != coPath {
		t.Errorf("ResolvePath with only CORALLINE_CONFIG = %q, want %q", got, coPath)
	}

	os.Unsetenv("CORALLINE_CONFIG")
	want := filepath.Join(configDir, "coralline.conf")
	if got := ResolvePath(configDir); got != want {
		t.Errorf("ResolvePath default = %q, want %q", got, want)
	}
}
