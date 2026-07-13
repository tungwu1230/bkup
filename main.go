package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tungwu1230/bkup/internal/archive"
	"github.com/tungwu1230/bkup/internal/ignore"
	"github.com/tungwu1230/bkup/internal/naming"
	"github.com/tungwu1230/bkup/internal/walker"
)

// version is overridden at release time by GoReleaser via
// -ldflags "-X main.version=...".
var version = "dev"

func main() {
	err := Run(os.Args[1:], os.Stdout)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// Run parses args and performs a backup, writing progress/summary to stdout.
func Run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("bkup", flag.ContinueOnError)
	fs.SetOutput(stdout)

	var outputDir string
	fs.StringVar(&outputDir, "o", ".", "output directory for the backup archive")
	fs.StringVar(&outputDir, "output", ".", "output directory for the backup archive")

	var verbose bool
	fs.BoolVar(&verbose, "v", false, "list files as they are packed")
	fs.BoolVar(&verbose, "verbose", false, "list files as they are packed")

	var showVersion bool
	fs.BoolVar(&showVersion, "version", false, "print version and exit")

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: bkup [-o output-dir] [-v] <folder>")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if showVersion {
		fmt.Fprintln(stdout, "bkup", version)
		return nil
	}
	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("missing required <folder> argument")
	}
	sourceDir := fs.Arg(0)

	m, err := ignore.Load(sourceDir)
	if err != nil {
		return fmt.Errorf("load .gitignore: %w", err)
	}

	entries, err := walker.Collect(sourceDir, m)
	if err != nil {
		return fmt.Errorf("collect files: %w", err)
	}

	archivePath := naming.OutputPath(sourceDir, outputDir, time.Now())
	if err := archive.Create(archivePath, sourceDir, entries); err != nil {
		return fmt.Errorf("create archive: %w", err)
	}

	if verbose {
		for _, e := range entries {
			fmt.Fprintln(stdout, e)
		}
	}
	fmt.Fprintf(stdout, "backed up %d entries to %s\n", len(entries), archivePath)
	return nil
}
