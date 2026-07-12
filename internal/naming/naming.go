package naming

import (
	"fmt"
	"path/filepath"
	"time"
)

// OutputPath returns the destination zip path for backing up sourceDir into
// outputDir, named "{folderName}_backup_{YYYYMMDD}.zip".
func OutputPath(sourceDir, outputDir string, now time.Time) string {
	folderName := filepath.Base(sourceDir)
	filename := fmt.Sprintf("%s_backup_%s.zip", folderName, now.Format("20060102"))
	return filepath.Join(outputDir, filename)
}
