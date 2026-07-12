package ignore_test

import (
	"os"
	"path/filepath"
	"testing"

	"bkup/internal/ignore"
)

func TestMatch_IgnoresFileMatchingPattern(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".gitignore"), "*.log\n")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !m.Match("app.log", false) {
		t.Errorf("Match(%q, false) = false, want true", "app.log")
	}
	if m.Match("main.go", false) {
		t.Errorf("Match(%q, false) = true, want false", "main.go")
	}
}

func TestMatch_DirectoryOnlyPatternDoesNotMatchFileWithSameName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".gitignore"), "node_modules/\n")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !m.Match("node_modules", true) {
		t.Errorf("Match(%q, true) = false, want true", "node_modules")
	}
	if m.Match("node_modules", false) {
		t.Errorf("Match(%q, false) = true, want false", "node_modules")
	}
}

func TestMatch_NegationPatternReincludesFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".gitignore"), "*.log\n!important.log\n")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if m.Match("important.log", false) {
		t.Errorf("Match(%q, false) = true, want false", "important.log")
	}
	if !m.Match("debug.log", false) {
		t.Errorf("Match(%q, false) = false, want true", "debug.log")
	}
}

func TestMatch_NoGitignoreFileIgnoresNothing(t *testing.T) {
	dir := t.TempDir()

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if m.Match("anything.txt", false) {
		t.Errorf("Match(%q, false) = true, want false", "anything.txt")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
