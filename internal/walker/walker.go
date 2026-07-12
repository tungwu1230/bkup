package walker

import (
	"os"
	"path/filepath"

	"bkup/internal/ignore"
)

// gitDir is always excluded, independent of .gitignore rules.
const gitDir = ".git"

// Collect walks rootDir and returns the "/"-separated relative paths of all
// regular files that are not excluded by m and are not inside .git.
func Collect(rootDir string, m *ignore.Matcher) ([]string, error) {
	var files []string

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
			return nil
		}

		if m.Match(rel, false) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
