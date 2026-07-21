package walker_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/tungwu1230/bkup/internal/ignore"
	"github.com/tungwu1230/bkup/internal/walker"
)

func TestCollect_ReturnsAllFilesWhenNothingIgnored(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	writeFile(t, filepath.Join(dir, "README.md"), "# hi")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("ignore.Load() error = %v", err)
	}

	got, err := walker.Collect(dir, m)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	sort.Strings(got)

	want := []string{"README.md", "main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect() = %v, want %v", got, want)
	}
}

func TestCollect_ExcludesFilesMatchingGitignoreAndIncludesNestedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".gitignore"), "*.log\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	writeFile(t, filepath.Join(dir, "debug.log"), "noisy")
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "src", "app.go"), "package src")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("ignore.Load() error = %v", err)
	}

	got, err := walker.Collect(dir, m)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	sort.Strings(got)

	want := []string{".gitignore", "main.go", "src", "src/app.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect() = %v, want %v", got, want)
	}
}

func TestCollect_IncludesEmptyDirectoriesAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink("main.go", filepath.Join(dir, "link.go")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("ignore.Load() error = %v", err)
	}

	got, err := walker.Collect(dir, m)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	sort.Strings(got)

	want := []string{"empty", "link.go", "main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect() = %v, want %v", got, want)
	}
}

func TestCollect_SkipsEntireIgnoredDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".gitignore"), "node_modules/\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "node_modules", "pkg", "index.js"), "console.log(1)")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("ignore.Load() error = %v", err)
	}

	got, err := walker.Collect(dir, m)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	sort.Strings(got)

	want := []string{".gitignore", "main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect() = %v, want %v", got, want)
	}
}

func TestCollect_AlwaysSkipsDotGitDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("ignore.Load() error = %v", err)
	}

	got, err := walker.Collect(dir, m)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	sort.Strings(got)

	want := []string{"main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect() = %v, want %v", got, want)
	}
}

func TestCollect_HonorsNestedGitignoreScopedToItsSubtree(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	writeFile(t, filepath.Join(dir, "debug.log"), "root log")
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "src", ".gitignore"), "*.tmp\n")
	writeFile(t, filepath.Join(dir, "src", "app.go"), "package src")
	writeFile(t, filepath.Join(dir, "src", "scratch.tmp"), "scratch")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("ignore.Load() error = %v", err)
	}

	got, err := walker.Collect(dir, m)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	sort.Strings(got)

	// *.tmp is only ignored inside src/ (where it's declared); debug.log at
	// the root isn't touched by src/'s .gitignore.
	want := []string{"debug.log", "main.go", "src", "src/.gitignore", "src/app.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect() = %v, want %v", got, want)
	}
}

func TestCollect_NestedGitignoreAddsToAncestorRulesRatherThanReplacingThem(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".gitignore"), "*.log\n")
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "src", ".gitignore"), "*.tmp\n")
	writeFile(t, filepath.Join(dir, "src", "app.go"), "package src")
	writeFile(t, filepath.Join(dir, "src", "debug.log"), "still ignored via root rule")
	writeFile(t, filepath.Join(dir, "src", "scratch.tmp"), "ignored via nested rule")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("ignore.Load() error = %v", err)
	}

	got, err := walker.Collect(dir, m)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	sort.Strings(got)

	want := []string{".gitignore", "src", "src/.gitignore", "src/app.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect() = %v, want %v", got, want)
	}
}

func TestCollect_NestedIgnoredDirectoryIsSkippedEntirely(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "src", ".gitignore"), "build/\n")
	writeFile(t, filepath.Join(dir, "src", "app.go"), "package src")
	if err := os.MkdirAll(filepath.Join(dir, "src", "build"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "src", "build", "out.bin"), "binary")

	m, err := ignore.Load(dir)
	if err != nil {
		t.Fatalf("ignore.Load() error = %v", err)
	}

	got, err := walker.Collect(dir, m)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	sort.Strings(got)

	want := []string{"src", "src/.gitignore", "src/app.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Collect() = %v, want %v", got, want)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
