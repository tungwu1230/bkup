package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_CreatesBackupArchiveRespectingGitignore(t *testing.T) {
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
	archiveName := entries[0].Name()
	if !strings.HasPrefix(archiveName, folderName+"_backup_") || !strings.HasSuffix(archiveName, ".tar.gz") {
		t.Errorf("archive name = %q, want prefix %q and suffix .tar.gz", archiveName, folderName+"_backup_")
	}

	names := readArchiveNames(t, filepath.Join(outDir, archiveName))
	if !names["main.go"] {
		t.Errorf("archive missing main.go, got %v", names)
	}
	if !names[".gitignore"] {
		t.Errorf("archive missing .gitignore, got %v", names)
	}
	if names["debug.log"] {
		t.Errorf("archive should not contain debug.log, got %v", names)
	}
}

func TestRun_HonorsNestedGitignore(t *testing.T) {
	srcDir := t.TempDir()
	mustWrite(t, filepath.Join(srcDir, ".gitignore"), "*.log\n")
	mustWrite(t, filepath.Join(srcDir, "main.go"), "package main")
	mustWrite(t, filepath.Join(srcDir, "debug.log"), "noisy")
	if err := os.MkdirAll(filepath.Join(srcDir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mustWrite(t, filepath.Join(srcDir, "src", ".gitignore"), "*.tmp\n")
	mustWrite(t, filepath.Join(srcDir, "src", "app.go"), "package src")
	mustWrite(t, filepath.Join(srcDir, "src", "scratch.tmp"), "scratch")
	mustWrite(t, filepath.Join(srcDir, "src", "trace.log"), "still ignored via root rule")

	outDir := t.TempDir()

	var stdout bytes.Buffer
	if err := Run([]string{"-o", outDir, srcDir}, &stdout); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in output dir, got %d: %v", len(entries), entries)
	}

	names := readArchiveNames(t, filepath.Join(outDir, entries[0].Name()))

	for _, want := range []string{"main.go", ".gitignore", "src/", "src/.gitignore", "src/app.go"} {
		if !names[want] {
			t.Errorf("archive missing %q, got %v", want, names)
		}
	}
	for _, notWant := range []string{"debug.log", "src/scratch.tmp", "src/trace.log"} {
		if names[notWant] {
			t.Errorf("archive should not contain %q, got %v", notWant, names)
		}
	}
}

func TestRun_LongFormFlagsWork(t *testing.T) {
	srcDir := t.TempDir()
	mustWrite(t, filepath.Join(srcDir, "main.go"), "package main")

	outDir := t.TempDir()

	var stdout bytes.Buffer
	if err := Run([]string{"--output", outDir, "--verbose", srcDir}, &stdout); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in output dir, got %d: %v", len(entries), entries)
	}

	if !strings.Contains(stdout.String(), "main.go") {
		t.Errorf("stdout = %q, want it to list main.go (verbose mode)", stdout.String())
	}
}

func TestRun_ReturnsErrorWhenFolderArgMissing(t *testing.T) {
	var stdout bytes.Buffer
	err := Run([]string{}, &stdout)
	if err == nil {
		t.Fatal("Run() error = nil, want error for missing folder argument")
	}
	if errors.Is(err, flag.ErrHelp) {
		t.Error("Run() error = flag.ErrHelp, want a distinct missing-argument error")
	}
	if !strings.Contains(stdout.String(), "<folder>") {
		t.Errorf("stdout = %q, want it to include usage mentioning <folder>", stdout.String())
	}
}

func TestRun_HelpFlagReturnsErrHelpAndPrintsFolderUsage(t *testing.T) {
	var stdout bytes.Buffer
	err := Run([]string{"--help"}, &stdout)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("Run() error = %v, want flag.ErrHelp", err)
	}
	if !strings.Contains(stdout.String(), "<folder>") {
		t.Errorf("stdout = %q, want usage text to mention <folder>", stdout.String())
	}
}

func TestRun_VersionFlagPrintsVersion(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run([]string{"--version"}, &stdout); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := stdout.String(), "bkup "+version+"\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func readArchiveNames(t *testing.T, archivePath string) map[string]bool {
	t.Helper()

	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer gr.Close()

	names := map[string]bool{}
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read entry: %v", err)
		}
		names[hdr.Name] = true
	}
	return names
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
