# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`bkup` is a Go CLI that backs up a folder into a `tar.gz` archive, the way `git archive` would — but without requiring a git repository, and including uncommitted files. It reads the folder's root `.gitignore`, walks the tree, and archives everything not excluded (always skipping `.git/`).

## Commands

```sh
go build ./...
go vet ./...
go test ./...
```

Run a single test:

```sh
go test ./internal/walker/ -run TestCollect
```

There is no separate lint config beyond `go vet`; CI (`.github/workflows/ci.yml`) runs build, vet, and test on both `ubuntu-latest` and `macos-latest`.

## Architecture

Pipeline in `main.go`'s `Run(args, stdout)` (the testable entry point; `main()` just wires it to `os.Args`/`os.Stdout` and sets the process exit code):

1. `internal/ignore.Load(sourceDir)` — reads `sourceDir/.gitignore` and returns a `Matcher`. Missing file → a `Matcher` that ignores nothing. `Load` is directory-agnostic, so `walker` also calls it per-directory to pick up nested `.gitignore` files.
2. `internal/walker.Collect(sourceDir, rootMatcher)` — `filepath.WalkDir`s the tree, maintaining a stack of `(dir, *ignore.Matcher)` frames (root plus one per directory with its own `.gitignore`) so nested `.gitignore` files apply only within their own subtree. A path is excluded if *any* frame on the stack matches it — a nested `.gitignore` can add exclusions but cannot `!`-negate something an ancestor `.gitignore` already excludes (mirrors git's own limitation for excluded directories). Unconditionally skips `.git/`. Returns `/`-separated relative paths for dirs, regular files, and symlinks (irregular entries like sockets/devices are skipped). Directories are included explicitly so empty dirs survive.
3. `internal/naming.OutputPath(sourceDir, outputDir, now)` — builds `{folderName}_backup_{YYYYMMDD}.tar.gz` under `outputDir`.
4. `internal/archive.Create(archivePath, sourceDir, entries)` — writes the entries into a gzip'd tar, preserving mtimes, permissions, and symlinks (stored as symlinks, not followed/dereferenced).

Each `internal/*` package is unit-tested against its own public API; `main_test.go` covers the end-to-end flow through `Run`.

Key behavioral decisions baked into the code (don't "fix" without discussion):
- Nested `.gitignore` files are additive only: a directory's `.gitignore` scopes to its own subtree and stacks with ancestors', but can't re-include a path an ancestor already excluded.
- Symlinks pointing outside the folder are preserved as symlinks rather than followed.
- Re-running on the same day overwrites the previous archive with the same name (filename is date-, not time-, granular).

## Release process

Version is injected via `-X main.version=...` (ldflags) by GoReleaser at release time; `version = "dev"` is the source default. Tagging `v*` triggers `.github/workflows/release.yml`, which runs GoReleaser (`.goreleaser.yaml`) to build cross-platform binaries and publish a Homebrew cask to `tungwu1230/homebrew-tap`.
