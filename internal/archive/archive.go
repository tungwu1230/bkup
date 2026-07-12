package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// Create writes a zip archive at zipPath containing each of files (given as
// "/"-separated paths relative to rootDir), preserving their relative paths
// as zip entry names.
func Create(zipPath, rootDir string, files []string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, rel := range files {
		if err := addFile(zw, rootDir, rel); err != nil {
			return err
		}
	}
	return nil
}

func addFile(zw *zip.Writer, rootDir, rel string) error {
	src, err := os.Open(filepath.Join(rootDir, filepath.FromSlash(rel)))
	if err != nil {
		return err
	}
	defer src.Close()

	w, err := zw.Create(rel)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, src)
	return err
}
