package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Create writes a gzip-compressed tar archive at archivePath containing each
// of entries (given as "/"-separated paths relative to rootDir), preserving
// their relative paths as entry names. Directories, regular files, and
// symlinks are supported; modification times and permission bits are
// preserved in the archive.
func Create(archivePath, rootDir string, entries []string) error {
	out, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	tw := tar.NewWriter(gw)

	for _, rel := range entries {
		if err := addEntry(tw, rootDir, rel); err != nil {
			return err
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}
	return out.Close()
}

func addEntry(tw *tar.Writer, rootDir, rel string) error {
	path := filepath.Join(rootDir, filepath.FromSlash(rel))

	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	link := ""
	if fi.Mode()&os.ModeSymlink != 0 {
		if link, err = os.Readlink(path); err != nil {
			return err
		}
	}

	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return err
	}
	hdr.Name = rel
	if fi.IsDir() {
		hdr.Name += "/"
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	if !fi.Mode().IsRegular() {
		return nil
	}

	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(tw, src)
	return err
}

// Extract reads a gzip-compressed tar archive at archivePath and writes its
// entries under destDir. Each entry's resolved path is verified to stay
// within destDir before anything is written; an entry that would escape
// destDir (e.g. via a "../" name) aborts the extraction with an error.
// Only directories, regular files, and symlinks are restored. A symlink
// entry is rejected if the target it would point to resolves outside
// destDir. Before writing a directory, regular file, or symlink, any
// pre-existing symlink already at that path is removed first, so an entry
// can never write through a symlink (whether planted by an earlier entry in
// the same archive or already present in destDir) to a location outside the
// extraction tree. Any other entry type is rejected.
func Extract(archivePath, destDir string) error {
	in, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer in.Close()

	gr, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := extractEntry(tr, destDir, hdr); err != nil {
			return err
		}
	}
	return nil
}

func extractEntry(tr *tar.Reader, destDir string, hdr *tar.Header) error {
	target := filepath.Join(destDir, filepath.FromSlash(hdr.Name))

	rel, err := filepath.Rel(destDir, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("archive entry %q escapes destination directory", hdr.Name)
	}

	mode := os.FileMode(hdr.Mode) & 0o777

	switch hdr.Typeflag {
	case tar.TypeDir:
		if err := removeExistingSymlink(target); err != nil {
			return err
		}
		return os.MkdirAll(target, mode)
	case tar.TypeReg:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := removeExistingSymlink(target); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, tr); err != nil {
			return err
		}
		return out.Close()
	case tar.TypeSymlink:
		linkTarget := hdr.Linkname
		if !filepath.IsAbs(linkTarget) {
			linkTarget = filepath.Join(filepath.Dir(target), linkTarget)
		}
		linkRel, err := filepath.Rel(destDir, linkTarget)
		if err != nil || linkRel == ".." || strings.HasPrefix(linkRel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("archive symlink %q points outside destination directory", hdr.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := removeExistingSymlink(target); err != nil {
			return err
		}
		return os.Symlink(hdr.Linkname, target)
	default:
		return fmt.Errorf("archive entry %q has unsupported type", hdr.Name)
	}
}

// removeExistingSymlink removes the symlink at target, if one exists, so a
// later write to that path cannot be redirected through it. Non-symlinks
// and nonexistent paths are left untouched.
func removeExistingSymlink(target string) error {
	fi, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	return os.Remove(target)
}
