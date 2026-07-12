package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
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
