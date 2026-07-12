package ignore

import (
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// Matcher decides whether a path relative to a loaded root directory is
// ignored, based on that directory's .gitignore file.
type Matcher struct {
	gi *gitignore.GitIgnore
}

// Load reads rootDir/.gitignore and returns a Matcher for it. If no
// .gitignore file exists, the returned Matcher ignores nothing.
func Load(rootDir string) (*Matcher, error) {
	data, err := os.ReadFile(filepath.Join(rootDir, ".gitignore"))
	if os.IsNotExist(err) {
		return &Matcher{}, nil
	}
	if err != nil {
		return nil, err
	}

	gi := gitignore.CompileIgnoreLines(strings.Split(string(data), "\n")...)
	return &Matcher{gi: gi}, nil
}

// Match reports whether relPath (relative to the loaded root, using "/"
// separators) is ignored.
func (m *Matcher) Match(relPath string, isDir bool) bool {
	if m.gi == nil {
		return false
	}
	if isDir && !strings.HasSuffix(relPath, "/") {
		relPath += "/"
	}
	return m.gi.MatchesPath(relPath)
}
