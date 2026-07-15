// Package gitstate runs the single, time-bounded git subprocess and parses its
// porcelain v2 output. It is a faithful port of statusline.sh's read_git for the
// fields the git segment consumes.
package gitstate

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Timeout bounds the git subprocess (design: 2.5-second timeout). It is a
// package variable so tests can shorten it.
var Timeout = 2500 * time.Millisecond

// State holds the parsed git status the git segment renders.
type State struct {
	Branch string // branch name, or short oid when detached
	Marks  string // staged '+', modified '!', untracked '?' in that order
	AB     string // ahead/behind, e.g. "⇡2⇣1"
	Dirty  bool   // true when any mark is set
	oid    string // branch.oid; empty means "not a repo"
}

// Present reports whether the cwd is inside a git repository (the git segment is
// hidden otherwise).
func (s State) Present() bool { return s.oid != "" }

// newGitCmd builds the git invocation. It is a variable so tests can substitute
// a stand-in command. GIT_OPTIONAL_LOCKS=0 stops a frequently-refreshed
// statusline from contending for index.lock (notably on Windows).
var newGitCmd = func(ctx context.Context, cwd string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", "-C", cwd, "status", "--porcelain=v2", "--branch")
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	return cmd
}

// Run invokes git with a hard timeout and returns the parsed State. On timeout
// the child process is killed (exec.CommandContext) and on any failure an empty
// State is returned, so the git segment is simply hidden. It never writes to
// stdout/stderr.
func Run(parent context.Context, cwd string) State {
	if cwd == "" {
		return State{}
	}
	ctx, cancel := context.WithTimeout(parent, Timeout)
	defer cancel()

	cmd := newGitCmd(ctx, cwd)
	out, err := cmd.Output()
	if err != nil {
		// Timeout, non-repo error, or missing git: hide the segment.
		return State{}
	}
	return Parse(string(out))
}

// Parse parses `git status --porcelain=v2 --branch` output. Empty/ownerless
// output (no branch.oid line) yields a State with Present()==false.
func Parse(output string) State {
	var (
		oid, head, aStr, bStr       string
		staged, unstaged, untracked bool
	)
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.oid "):
			oid = strings.TrimPrefix(line, "# branch.oid ")
		case strings.HasPrefix(line, "# branch.head "):
			head = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.ab "):
			fields := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
			if len(fields) >= 2 {
				aStr = strings.TrimPrefix(fields[0], "+")
				bStr = strings.TrimPrefix(fields[1], "-")
			}
		case strings.HasPrefix(line, "? "):
			untracked = true
		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 "):
			// Ordinary/renamed change: after the "N " prefix, the two XY status
			// chars indicate staged (X) and unstaged (Y); '.' means unchanged.
			xy := line[2:]
			if len(xy) >= 1 && xy[0] != '.' {
				staged = true
			}
			if len(xy) >= 2 && xy[1] != '.' {
				unstaged = true
			}
		case strings.HasPrefix(line, "u "):
			unstaged = true
		}
	}

	if oid == "" {
		return State{} // not a repo
	}

	st := State{oid: oid}
	if head == "(detached)" || head == "" {
		if len(oid) >= 7 {
			st.Branch = oid[:7]
		} else {
			st.Branch = oid
		}
	} else {
		st.Branch = head
	}

	if staged {
		st.Marks += "+"
	}
	if unstaged {
		st.Marks += "!"
	}
	if untracked {
		st.Marks += "?"
	}
	if a, err := strconv.Atoi(aStr); err == nil && a > 0 {
		st.AB += "⇡" + strconv.Itoa(a)
	}
	if b, err := strconv.Atoi(bStr); err == nil && b > 0 {
		st.AB += "⇣" + strconv.Itoa(b)
	}
	st.Dirty = st.Marks != ""
	return st
}
