package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_CreatesBackupZipRespectingGitignore(t *testing.T) {
	srcDir := t.TempDir()
	mustWrite(t, filepath.Join(srcDir, ".gitignore"), "*.log\n")
	mustWrite(t, filepath.Join(srcDir, "main.go"), "package main")
	mustWrite(t, filepath.Join(srcDir, "debug.log"), "noisy")

	outDir := t.TempDir()

	var stdout bytes.Buffer
	if err := Run([]string{"-o", outDir, srcDir}, &stdout); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	folderName := filepath.Base(srcDir)
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in output dir, got %d: %v", len(entries), entries)
	}
	zipName := entries[0].Name()
	if !strings.HasPrefix(zipName, folderName+"_backup_") || !strings.HasSuffix(zipName, ".zip") {
		t.Errorf("zip name = %q, want prefix %q and suffix .zip", zipName, folderName+"_backup_")
	}

	r, err := zip.OpenReader(filepath.Join(outDir, zipName))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	defer r.Close()

	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	if !names["main.go"] {
		t.Errorf("zip missing main.go, got %v", names)
	}
	if !names[".gitignore"] {
		t.Errorf("zip missing .gitignore, got %v", names)
	}
	if names["debug.log"] {
		t.Errorf("zip should not contain debug.log, got %v", names)
	}
}

func TestRun_ReturnsErrorWhenFolderArgMissing(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run([]string{}, &stdout); err == nil {
		t.Error("Run() error = nil, want error for missing folder argument")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
