// Package conf loads coralline configuration, compatible with the bash
// implementation's generated config subset (see design: "conf 相容解析器只支援生成子集").
//
// The parser supports exactly the syntax that configure.sh emits: comment and
// blank lines, `VAR=value` / `VAR="value"` / `VAR='value'` assignments, and a
// theme-source line `. "$_VL_DIR/themes/<name>.conf"`. Everything else is
// silently ignored, matching bash's behavior of not affecting output for lines
// the renderer does not consume.
package conf

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds resolved configuration values keyed by their VL_* variable name.
type Config struct {
	values map[string]string
}

// Get returns the value for key, or "" if unset.
func (c *Config) Get(key string) string {
	if c == nil || c.values == nil {
		return ""
	}
	return c.values[key]
}

// GetInt returns the value for key parsed as int, or def when the value is
// empty or not a number.
func (c *Config) GetInt(key string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(c.Get(key))); err == nil {
		return n
	}
	return def
}

// set records an assignment (later assignments overwrite earlier ones).
func (c *Config) set(key, val string) {
	c.values[key] = val
}

// Set overrides a value programmatically (used by callers and tests).
func (c *Config) Set(key, val string) {
	c.values[key] = val
}

// ResolvePath returns the config file path. The VL_CONFIG environment variable
// (spec) takes precedence; CORALLINE_CONFIG (the bash implementation's variable)
// is honored as a fallback; otherwise the default is <configDir>/coralline.conf,
// matching bash's `$_VL_CONFIG_DIR/coralline.conf` (configDir is the parent of
// the renderer executable's directory). If configDir is empty, it falls back to
// ~/.claude/coralline.conf.
func ResolvePath(configDir string) string {
	if p := os.Getenv("VL_CONFIG"); p != "" {
		return p
	}
	if p := os.Getenv("CORALLINE_CONFIG"); p != "" {
		return p
	}
	if configDir != "" {
		return filepath.Join(configDir, "coralline.conf")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "coralline.conf"
	}
	return filepath.Join(home, ".claude", "coralline.conf")
}

// Defaults returns a Config populated with the built-in default values, mirroring
// the defaults block of statusline.sh.
func Defaults() *Config {
	c := &Config{values: map[string]string{}}
	for k, v := range defaults {
		c.values[k] = v
	}
	return c
}

// Load returns Defaults() overlaid with the assignments parsed from path. A
// missing file is not an error; defaults are returned unchanged.
//
// vlDir is the value of $_VL_DIR used to expand theme-source lines: the
// directory of the renderer executable, matching bash's `_VL_DIR="${0%/*}"`
// (statusline.sh's own directory). In the real deployment that is
// ~/.claude/coralline, one level below the config file at ~/.claude. When vlDir
// is empty it falls back to the config file's own directory (used by tests and
// callers with no theme-source lines).
func Load(path, vlDir string) (*Config, error) {
	c := Defaults()
	if path == "" {
		return c, nil
	}
	if _, err := os.Stat(path); err != nil {
		// Missing config file: bash guards with `[ -f "$VL_CONF" ]`, so this is
		// not an error — defaults stand.
		return c, nil
	}
	if vlDir == "" {
		vlDir = filepath.Dir(path)
	}
	// Use an absolute path so an expanded source target is recognized as
	// already-resolved and not joined onto vlDir a second time.
	if abs, err := filepath.Abs(vlDir); err == nil {
		vlDir = abs
	}
	if err := c.parseFile(path, vlDir); err != nil {
		return c, err
	}
	return c, nil
}

// parseFile applies the assignments in path onto c. Theme-source lines are
// expanded in place by parsing the referenced file. vlDir is the value of
// $_VL_DIR (the root config file's directory).
func (c *Config) parseFile(path, vlDir string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		c.parseLine(sc.Text(), vlDir)
	}
	return sc.Err()
}

func (c *Config) parseLine(line, vlDir string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return
	}
	// Theme-source line: `. "$_VL_DIR/themes/<name>.conf"` (also accept `source`).
	if rest, ok := sourceTarget(trimmed); ok {
		target := expandVLDir(rest, vlDir)
		if !filepath.IsAbs(target) {
			target = filepath.Join(vlDir, target)
		}
		// A missing theme file is silently ignored (bash `.` would error to
		// stderr but not affect stdout); parse it in place when present.
		if _, err := os.Stat(target); err == nil {
			_ = c.parseFile(target, vlDir)
		}
		return
	}
	if key, val, ok := assignment(trimmed); ok {
		c.set(key, val)
	}
	// Anything else: silently ignored.
}

// sourceTarget reports whether line is a `.`/`source` directive and returns the
// (still-quoted, unexpanded) target argument.
func sourceTarget(line string) (string, bool) {
	var rest string
	switch {
	case strings.HasPrefix(line, ". "):
		rest = strings.TrimSpace(line[2:])
	case strings.HasPrefix(line, "source "):
		rest = strings.TrimSpace(line[len("source "):])
	default:
		return "", false
	}
	// The target must be a single quoted or bare token; a compound line like
	// `[ -f x ] && . "$VL_CONF"` never starts with `. ` so it does not reach here.
	return parseValue(rest), true
}

// assignment parses `VAR=value` with optional single/double quotes around the
// value. It returns ok=false for lines that are not a bare `VAR=` assignment.
func assignment(line string) (key, val string, ok bool) {
	eq := strings.IndexByte(line, '=')
	if eq <= 0 {
		return "", "", false
	}
	key = line[:eq]
	if !validVarName(key) {
		return "", "", false
	}
	val = parseValue(line[eq+1:])
	return key, val, true
}

// parseValue extracts a shell assignment's value: a single- or double-quoted
// string (contents returned, surrounding quotes dropped) or a bare token, with a
// trailing `# comment` and surrounding whitespace ignored. This matches the
// generated-config subset; it does not implement shell quoting in general.
func parseValue(s string) string {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return ""
	}
	switch s[0] {
	case '"':
		if i := strings.IndexByte(s[1:], '"'); i >= 0 {
			return s[1 : 1+i]
		}
		return s[1:] // unterminated quote: take the rest
	case '\'':
		if i := strings.IndexByte(s[1:], '\''); i >= 0 {
			return s[1 : 1+i]
		}
		return s[1:]
	default:
		for i := 0; i < len(s); i++ {
			if s[i] == ' ' || s[i] == '\t' || s[i] == '#' {
				return s[:i]
			}
		}
		return s
	}
}

// validVarName reports whether name is a shell variable name: [A-Za-z_][A-Za-z0-9_]*.
func validVarName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		ch := name[i]
		switch {
		case ch == '_':
		case ch >= 'A' && ch <= 'Z':
		case ch >= 'a' && ch <= 'z':
		case i > 0 && ch >= '0' && ch <= '9':
		default:
			return false
		}
	}
	return true
}

// expandVLDir replaces occurrences of $_VL_DIR / ${_VL_DIR} with vlDir.
func expandVLDir(s, vlDir string) string {
	s = strings.ReplaceAll(s, "${_VL_DIR}", vlDir)
	s = strings.ReplaceAll(s, "$_VL_DIR", vlDir)
	return s
}
