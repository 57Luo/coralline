package gitstate

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestParseDirtyRepo(t *testing.T) {
	out := "# branch.oid abcdef1234567890abcdef1234567890abcdef12\n" +
		"# branch.head main\n" +
		"# branch.ab +2 -1\n" +
		"1 M. N... 100644 100644 100644 aaa bbb staged.txt\n" +
		"1 .M N... 100644 100644 100644 ccc ddd modified.txt\n" +
		"? untracked.txt\n"
	st := Parse(out)
	if st.Branch != "main" {
		t.Errorf("Branch = %q, want main", st.Branch)
	}
	if st.Marks != "+!?" {
		t.Errorf("Marks = %q, want +!?", st.Marks)
	}
	if st.AB != "⇡2⇣1" {
		t.Errorf("AB = %q, want ⇡2⇣1", st.AB)
	}
	if !st.Dirty {
		t.Errorf("Dirty = false, want true")
	}
	if !st.Present() {
		t.Errorf("Present() = false, want true")
	}
}

func TestParseCleanRepo(t *testing.T) {
	out := "# branch.oid abcdef1234567890abcdef1234567890abcdef12\n" +
		"# branch.head feature/x\n" +
		"# branch.ab +0 -0\n"
	st := Parse(out)
	if st.Branch != "feature/x" {
		t.Errorf("Branch = %q, want feature/x", st.Branch)
	}
	if st.Marks != "" {
		t.Errorf("Marks = %q, want empty", st.Marks)
	}
	if st.AB != "" {
		t.Errorf("AB = %q, want empty", st.AB)
	}
	if st.Dirty {
		t.Errorf("Dirty = true, want false")
	}
}

func TestParseDetachedHead(t *testing.T) {
	out := "# branch.oid abcdef1234567890abcdef1234567890abcdef12\n" +
		"# branch.head (detached)\n"
	st := Parse(out)
	if st.Branch != "abcdef1" {
		t.Errorf("Branch = %q, want abcdef1 (short oid)", st.Branch)
	}
}

func TestParseUnmergedCountsUnstaged(t *testing.T) {
	out := "# branch.oid abcdef1234567890abcdef1234567890abcdef12\n" +
		"# branch.head main\n" +
		"u UU N... 100644 100644 100644 100644 a b c d conflict.txt\n"
	st := Parse(out)
	if st.Marks != "!" {
		t.Errorf("Marks = %q, want ! (unmerged is unstaged)", st.Marks)
	}
	if !st.Dirty {
		t.Errorf("Dirty = false, want true")
	}
}

func TestParseNotARepo(t *testing.T) {
	st := Parse("")
	if st.Present() {
		t.Errorf("Present() = true for empty output, want false")
	}
	if st.Branch != "" {
		t.Errorf("Branch = %q, want empty", st.Branch)
	}
}

// TestHelperProcess is not a real test; it is re-executed as a stand-in for a
// hung git that sleeps well past the timeout.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

func TestRunTimeoutKillsChildAndHidesSegment(t *testing.T) {
	orig := newGitCmd
	origTimeout := Timeout
	t.Cleanup(func() { newGitCmd = orig; Timeout = origTimeout })

	Timeout = 300 * time.Millisecond
	newGitCmd = func(ctx context.Context, cwd string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	}

	start := time.Now()
	st := Run(context.Background(), "/whatever")
	elapsed := time.Since(start)

	if st.Present() {
		t.Errorf("Present() = true on timeout, want false (segment hidden)")
	}
	// Must return shortly after the timeout, not after the 30s sleep — proving
	// the child was killed rather than waited on.
	if elapsed > 3*time.Second {
		t.Errorf("Run took %v, expected ~%v (child not killed promptly)", elapsed, Timeout)
	}
}
