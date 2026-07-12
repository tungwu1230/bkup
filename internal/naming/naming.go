package naming

import (
	"fmt"
	"path/filepath"
	"time"
)

// OutputPath returns the destination archive path for backing up sourceDir
// into outputDir, named "{folderName}_backup_{YYYYMMDD}.tar.gz".
func OutputPath(sourceDir, outputDir string, now time.Time) string {
	folderName := filepath.Base(sourceDir)
	filename := fmt.Sprintf("%s_backup_%s.tar.gz", folderName, now.Format("20060102"))
	return filepath.Join(outputDir, filename)
}
