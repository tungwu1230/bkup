package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"backup-cli/internal/archive"
	"backup-cli/internal/ignore"
	"backup-cli/internal/naming"
	"backup-cli/internal/walker"
)

func main() {
	if err := Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// Run parses args and performs a backup, writing progress/summary to stdout.
func Run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("backup-cli", flag.ContinueOnError)

	outputDir := "."
	fs.StringVar(&outputDir, "o", ".", "output directory for the backup zip")
	fs.StringVar(&outputDir, "output", ".", "output directory for the backup zip")

	verbose := false
	fs.BoolVar(&verbose, "v", false, "list files as they are packed")
	fs.BoolVar(&verbose, "verbose", false, "list files as they are packed")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: backup-cli [-o output-dir] [-v] <folder>")
	}
	sourceDir := fs.Arg(0)

	m, err := ignore.Load(sourceDir)
	if err != nil {
		return fmt.Errorf("load .gitignore: %w", err)
	}

	files, err := walker.Collect(sourceDir, m)
	if err != nil {
		return fmt.Errorf("collect files: %w", err)
	}

	zipPath := naming.OutputPath(sourceDir, outputDir, time.Now())
	if err := archive.Create(zipPath, sourceDir, files); err != nil {
		return fmt.Errorf("create archive: %w", err)
	}

	if verbose {
		for _, f := range files {
			fmt.Fprintln(stdout, f)
		}
	}
	fmt.Fprintf(stdout, "backed up %d files to %s\n", len(files), zipPath)
	return nil
}
