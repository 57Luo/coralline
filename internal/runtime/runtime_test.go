package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestDetectNodePinFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".nvmrc", "v20.11.0\n")

	if got := DetectNode(dir, false); got != "20.11.0" {
		t.Errorf("DetectNode() = %q, want %q", got, "20.11.0")
	}
}

func TestDetectNodeNoFile(t *testing.T) {
	dir := t.TempDir()

	if got := DetectNode(dir, false); got != "" {
		t.Errorf("DetectNode() = %q, want empty", got)
	}
}

func TestDetectNodeVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".node-version", "18.17.1\n")

	if got := DetectNode(dir, false); got != "18.17.1" {
		t.Errorf("DetectNode() = %q, want %q", got, "18.17.1")
	}
}

func TestDetectPythonVirtualEnv(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "/home/user/.venvs/myenv")
	dir := t.TempDir()

	if got := DetectPython(dir, false); got != "myenv" {
		t.Errorf("DetectPython() = %q, want %q", got, "myenv")
	}
}

func TestDetectPythonCondaBase(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_DEFAULT_ENV", "base")
	dir := t.TempDir()

	if got := DetectPython(dir, false); got != "" {
		t.Errorf("DetectPython() = %q, want empty", got)
	}
}

func TestDetectPythonCondaNonBase(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_DEFAULT_ENV", "myproject")
	dir := t.TempDir()

	if got := DetectPython(dir, false); got != "myproject" {
		t.Errorf("DetectPython() = %q, want %q", got, "myproject")
	}
}

func TestDetectPythonPinFile(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_DEFAULT_ENV", "")
	dir := t.TempDir()
	writeFile(t, dir, ".python-version", "3.11.5\n")

	if got := DetectPython(dir, false); got != "3.11.5" {
		t.Errorf("DetectPython() = %q, want %q", got, "3.11.5")
	}
}

func TestDetectPythonNoDetection(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_DEFAULT_ENV", "")
	dir := t.TempDir()

	if got := DetectPython(dir, false); got != "" {
		t.Errorf("DetectPython() = %q, want empty", got)
	}
}
