package naming_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tungwu1230/bkup/internal/naming"
)

func TestOutputPath_BuildsFilenameFromFolderNameAndDate(t *testing.T) {
	sourceDir := "/Users/alice/projects/myapp"
	outputDir := "/Users/alice/backups"
	now := time.Date(2026, 7, 13, 10, 30, 0, 0, time.UTC)

	got := naming.OutputPath(sourceDir, outputDir, now)

	want := filepath.Join(outputDir, "myapp_backup_20260713.tar.gz")
	if got != want {
		t.Errorf("OutputPath() = %q, want %q", got, want)
	}
}
