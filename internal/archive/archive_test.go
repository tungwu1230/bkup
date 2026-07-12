package archive_test

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"backup-cli/internal/archive"
)

func TestCreate_WritesFilesWithRelativePathsAndContent(t *testing.T) {
	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "main.go"), "package main")
	if err := os.MkdirAll(filepath.Join(srcDir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(srcDir, "src", "app.go"), "package src")

	zipPath := filepath.Join(t.TempDir(), "out.zip")
	files := []string{"main.go", "src/app.go"}

	if err := archive.Create(zipPath, srcDir, files); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	defer r.Close()

	got := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read entry %s: %v", f.Name, err)
		}
		got[f.Name] = string(data)
	}

	want := map[string]string{
		"main.go":    "package main",
		"src/app.go": "package src",
	}
	for name, content := range want {
		if got[name] != content {
			t.Errorf("entry %q = %q, want %q", name, got[name], content)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d entries, want %d: %v", len(got), len(want), got)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
