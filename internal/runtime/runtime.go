// Package runtime detects the active Node.js and Python environment for a
// directory, mirroring statusline.sh's runtime_node and runtime_python.
package runtime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const probeTimeout = 2 * time.Second

// firstLine reads the first line of path and returns it trimmed of
// surrounding whitespace. It returns "" if the file cannot be read or is
// empty.
func firstLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(data), "\n")
	return strings.TrimSpace(line)
}

// walkUp calls check for dir and each of its ancestors up to (but not
// including) the filesystem root, stopping as soon as check returns a
// non-empty string.
func walkUp(start string, check func(dir string) string) string {
	dir := start
	for dir != "" && dir != "/" {
		if v := check(dir); v != "" {
			return v
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// DetectNode returns the active Node.js version label for cwd: the version
// pinned by a .nvmrc or .node-version file found by walking up from cwd, or,
// if probe is true and no pin file was found, the version reported by
// `node --version`. It returns "" if nothing is detected.
func DetectNode(cwd string, probe bool) string {
	if v := walkUp(cwd, func(dir string) string {
		for _, f := range []string{".nvmrc", ".node-version"} {
			path := filepath.Join(dir, f)
			if info, err := os.Stat(path); err != nil || info.IsDir() {
				continue
			}
			if v := firstLine(path); v != "" {
				return strings.TrimPrefix(v, "v")
			}
		}
		return ""
	}); v != "" {
		return v
	}

	if probe {
		ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "node", "--version").Output()
		if err == nil {
			v := strings.TrimSpace(string(out))
			if v != "" {
				return strings.TrimPrefix(v, "v")
			}
		}
	}
	return ""
}

// DetectPython returns the active Python env/version label for cwd: the
// VIRTUAL_ENV basename, the non-"base" CONDA_DEFAULT_ENV, the version pinned
// by a .python-version file found by walking up from cwd, or, if probe is
// true and nothing else was found, the version reported by
// `python3 --version`. It returns "" if nothing is detected.
func DetectPython(cwd string, probe bool) string {
	if v := os.Getenv("VIRTUAL_ENV"); v != "" {
		return filepath.Base(v)
	}
	// conda auto-activates `base` for most users, so it is not a meaningful "env".
	if v := os.Getenv("CONDA_DEFAULT_ENV"); v != "" && v != "base" {
		return v
	}

	if v := walkUp(cwd, func(dir string) string {
		path := filepath.Join(dir, ".python-version")
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			return ""
		}
		return firstLine(path)
	}); v != "" {
		return v
	}

	if probe {
		ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "python3", "--version").CombinedOutput()
		if err == nil {
			v := strings.TrimSpace(strings.TrimPrefix(string(out), "Python "))
			if v != "" {
				return v
			}
		}
	}
	return ""
}
