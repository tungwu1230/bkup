package archive_test

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tungwu1230/bkup/internal/archive"
)

func TestCreate_WritesFilesWithRelativePathsAndContent(t *testing.T) {
	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "main.go"), "package main")
	if err := os.MkdirAll(filepath.Join(srcDir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(srcDir, "src", "app.go"), "package src")

	archivePath := filepath.Join(t.TempDir(), "out.tar.gz")
	entries := []string{"main.go", "src", "src/app.go"}

	if err := archive.Create(archivePath, srcDir, entries); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got := readArchive(t, archivePath)

	want := map[string]string{
		"main.go":    "package main",
		"src/":       "",
		"src/app.go": "package src",
	}
	for name, content := range want {
		entry, ok := got[name]
		if !ok {
			t.Errorf("archive missing entry %q, got %v", name, got)
			continue
		}
		if entry.content != content {
			t.Errorf("entry %q = %q, want %q", name, entry.content, content)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d entries, want %d: %v", len(got), len(want), got)
	}
}

func TestCreate_PreservesModTimeAndPermissions(t *testing.T) {
	srcDir := t.TempDir()
	path := filepath.Join(srcDir, "script.sh")
	writeFile(t, path, "#!/bin/sh")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	mtime := time.Date(2023, 4, 5, 6, 7, 8, 0, time.UTC)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "out.tar.gz")
	if err := archive.Create(archivePath, srcDir, []string{"script.sh"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	entry, ok := readArchive(t, archivePath)["script.sh"]
	if !ok {
		t.Fatal("archive missing entry script.sh")
	}
	if !entry.hdr.ModTime.Equal(mtime) {
		t.Errorf("ModTime = %v, want %v", entry.hdr.ModTime, mtime)
	}
	if perm := entry.hdr.FileInfo().Mode().Perm(); perm != 0o755 {
		t.Errorf("Mode().Perm() = %o, want 755", perm)
	}
}

func TestCreate_PreservesSymlinksWithoutFollowingThem(t *testing.T) {
	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "real.txt"), "target")
	if err := os.Symlink("real.txt", filepath.Join(srcDir, "link.txt")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "out.tar.gz")
	entries := []string{"link.txt", "real.txt"}
	if err := archive.Create(archivePath, srcDir, entries); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	entry, ok := readArchive(t, archivePath)["link.txt"]
	if !ok {
		t.Fatal("archive missing entry link.txt")
	}
	if entry.hdr.Typeflag != tar.TypeSymlink {
		t.Errorf("Typeflag = %v, want tar.TypeSymlink", entry.hdr.Typeflag)
	}
	if entry.hdr.Linkname != "real.txt" {
		t.Errorf("Linkname = %q, want %q", entry.hdr.Linkname, "real.txt")
	}
}

func TestExtract_WritesFilesAndPreservesSymlinks(t *testing.T) {
	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "main.go"), "package main")
	if err := os.MkdirAll(filepath.Join(srcDir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(srcDir, "src", "app.go"), "package src")
	if err := os.Symlink("app.go", filepath.Join(srcDir, "src", "link.go")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "out.tar.gz")
	entries := []string{"main.go", "src", "src/app.go", "src/link.go"}
	if err := archive.Create(archivePath, srcDir, entries); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	destDir := t.TempDir()
	if err := archive.Extract(archivePath, destDir); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(destDir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if string(got) != "package main" {
		t.Errorf("main.go content = %q, want %q", got, "package main")
	}

	link, err := os.Readlink(filepath.Join(destDir, "src", "link.go"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if link != "app.go" {
		t.Errorf("link target = %q, want %q", link, "app.go")
	}
}

func TestExtract_RejectsPathTraversal(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "evil.tar.gz")
	writeMaliciousArchive(t, archivePath, "../evil.txt", "pwned")

	destDir := t.TempDir()
	if err := archive.Extract(archivePath, destDir); err == nil {
		t.Fatal("Extract() error = nil, want error for path-traversing entry")
	}

	escaped := filepath.Join(filepath.Dir(destDir), "evil.txt")
	if _, err := os.Stat(escaped); !os.IsNotExist(err) {
		t.Errorf("path-traversal entry was written outside destDir at %s", escaped)
	}
}

func TestExtract_RejectsSymlinkTargetEscapingDestDir(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "evil.tar.gz")
	writeMaliciousSymlink(t, archivePath, "link", "../../secret.txt")

	destDir := t.TempDir()
	if err := archive.Extract(archivePath, destDir); err == nil {
		t.Fatal("Extract() error = nil, want error for symlink escaping destDir")
	}

	if _, err := os.Lstat(filepath.Join(destDir, "link")); !os.IsNotExist(err) {
		t.Errorf("escaping symlink entry was written into destDir")
	}
}

func TestExtract_DoesNotWriteThroughPreexistingSymlink(t *testing.T) {
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.txt")
	writeFile(t, outsidePath, "original")

	destDir := t.TempDir()
	if err := os.Symlink(outsidePath, filepath.Join(destDir, "config")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "out.tar.gz")
	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "config"), "safe")
	if err := archive.Create(archivePath, srcDir, []string{"config"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := archive.Extract(archivePath, destDir); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	outsideContent, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside file: %v", err)
	}
	if string(outsideContent) != "original" {
		t.Errorf("outside file was written through symlink, content = %q", outsideContent)
	}

	fi, err := os.Lstat(filepath.Join(destDir, "config"))
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("destDir/config is still a symlink, want regular file")
	}
	insideContent, err := os.ReadFile(filepath.Join(destDir, "config"))
	if err != nil {
		t.Fatalf("read destDir/config: %v", err)
	}
	if string(insideContent) != "safe" {
		t.Errorf("destDir/config content = %q, want %q", insideContent, "safe")
	}
}

func writeMaliciousSymlink(t *testing.T, path, name, linkname string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name:     name,
		Mode:     0o777,
		Typeflag: tar.TypeSymlink,
		Linkname: linkname,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
}

func writeMaliciousArchive(t *testing.T, path, name, content string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name:     name,
		Mode:     0o644,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatalf("write content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
}

type archiveEntry struct {
	hdr     *tar.Header
	content string
}

func readArchive(t *testing.T, path string) map[string]archiveEntry {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer gr.Close()

	got := map[string]archiveEntry{}
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read entry: %v", err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read entry %s: %v", hdr.Name, err)
		}
		got[hdr.Name] = archiveEntry{hdr: hdr, content: string(data)}
	}
	return got
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
