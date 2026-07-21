package walker

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tungwu1230/bkup/internal/ignore"
)

// gitDir is always excluded, independent of .gitignore rules.
const gitDir = ".git"

// frame pairs a matcher with the "/"-separated directory (relative to
// rootDir) it was loaded from; dir is "" for the root itself.
type frame struct {
	dir string
	m   *ignore.Matcher
}

// Collect walks rootDir and returns the "/"-separated relative paths of all
// entries (directories, regular files, and symlinks) that are not excluded
// by rootMatcher or by any nested .gitignore encountered along the way, and
// are not inside .git. Directories are included so that empty directories
// and directory metadata survive the backup. Irregular entries such as
// sockets and device files are skipped.
//
// A nested .gitignore's patterns are scoped to its own subtree and stack
// with its ancestors' patterns, but it cannot re-include ("!"-negate) a path
// an ancestor .gitignore already excludes — the same limitation git itself
// documents for excluded directories, extended here to individual files for
// simplicity.
func Collect(rootDir string, rootMatcher *ignore.Matcher) ([]string, error) {
	var entries []string
	stack := []frame{{dir: "", m: rootMatcher}}

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

		// Pop frames belonging to subtrees we've backtracked out of. Since
		// WalkDir is a pre-order DFS that fully finishes a directory's
		// subtree before moving to its next sibling, rel's own depth always
		// matches the number of ancestor frames still relevant.
		if depth := strings.Count(rel, "/") + 1; len(stack) > depth {
			stack = stack[:depth]
		}

		if d.IsDir() {
			if d.Name() == gitDir {
				return filepath.SkipDir
			}
			if excluded(stack, rel, true) {
				return filepath.SkipDir
			}
			entries = append(entries, rel)

			m, err := ignore.Load(path)
			if err != nil {
				return err
			}
			stack = append(stack, frame{dir: rel, m: m})
			return nil
		}

		if !d.Type().IsRegular() && d.Type()&os.ModeSymlink == 0 {
			return nil
		}
		if excluded(stack, rel, false) {
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

// excluded reports whether rel is matched by any matcher on the stack, each
// checked against rel relative to that matcher's own directory.
func excluded(stack []frame, rel string, isDir bool) bool {
	for _, f := range stack {
		r := rel
		if f.dir != "" {
			r = strings.TrimPrefix(rel, f.dir+"/")
		}
		if f.m.Match(r, isDir) {
			return true
		}
	}
	return false
}
