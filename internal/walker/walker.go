package walker

import (
	"os"
	"path/filepath"

	"github.com/tungwu1230/bkup/internal/ignore"
)

// gitDir is always excluded, independent of .gitignore rules.
const gitDir = ".git"

// Collect walks rootDir and returns the "/"-separated relative paths of all
// entries (directories, regular files, and symlinks) that are not excluded
// by m and are not inside .git. Directories are included so that empty
// directories and directory metadata survive the backup. Irregular entries
// such as sockets and device files are skipped.
func Collect(rootDir string, m *ignore.Matcher) ([]string, error) {
	var entries []string

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootDir {
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if d.Name() == gitDir {
				return filepath.SkipDir
			}
			if m.Match(rel, true) {
				return filepath.SkipDir
			}
			entries = append(entries, rel)
			return nil
		}

		if !d.Type().IsRegular() && d.Type()&os.ModeSymlink == 0 {
			return nil
		}
		if m.Match(rel, false) {
			return nil
		}
		entries = append(entries, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}
