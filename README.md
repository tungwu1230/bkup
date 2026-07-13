# bkup

[![CI](https://github.com/tungwu1230/bkup/actions/workflows/ci.yml/badge.svg)](https://github.com/tungwu1230/bkup/actions/workflows/ci.yml)

A CLI that backs up a folder into a tar.gz archive, the way `git archive`
would — but it doesn't require the folder to be a git repository, and it
includes uncommitted files.

It reads the folder's root `.gitignore`, walks the tree, and archives
everything that isn't excluded (also skipping `.git/` unconditionally).
Modification times, permission bits, symlinks, and empty directories are
all preserved.

## Install

### With Homebrew (macOS)

```sh
brew install tungwu1230/tap/bkup
```

### With Go

Requires Go 1.26+.

```sh
go install github.com/tungwu1230/bkup@latest
```

This builds the binary into `$(go env GOPATH)/bin/bkup`. Make sure that
directory is on your `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Download a release binary

No Go toolchain required. Grab the archive for your OS/architecture from
the [releases page](https://github.com/tungwu1230/bkup/releases), verify
it against `checksums.txt`, and put the binary anywhere on your `PATH`:

```sh
tar -xzf bkup_<version>_<os>_<arch>.tar.gz
mv bkup /usr/local/bin/
```

For example, on an Apple Silicon Mac:

```sh
shasum -a 256 -c --ignore-missing checksums.txt
tar -xzf bkup_0.1.0_darwin_arm64.tar.gz
mv bkup /usr/local/bin/
```

(Windows releases are `.zip` archives containing `bkup.exe`.)

## Usage

```
bkup [-o output-dir] [-v] <folder>
```

| Flag | Default | Description |
|---|---|---|
| `-o`, `--output` | `.` (current directory) | Directory to write the archive into |
| `-v`, `--verbose` | off | List each packed entry to stdout |

The output file is named `{folderName}_backup_{YYYYMMDD}.tar.gz`.

### Example

```sh
$ bkup -o ~/backups ~/projects/myapp
backed up 42 entries to /Users/you/backups/myapp_backup_20260713.tar.gz
```

Running it with no `-o` writes the archive into the current directory:

```sh
$ cd ~/Desktop
$ bkup myapp
backed up 42 entries to myapp_backup_20260713.tar.gz
```

To restore a backup:

```sh
tar -xzf myapp_backup_20260713.tar.gz -C <destination>
```

## Behavior notes

- Only the source folder's **root** `.gitignore` is honored — nested
  `.gitignore` files in subdirectories are not read. This is a deliberate
  scope decision, not a limitation of the underlying matcher.
- `.git/` is always excluded, regardless of `.gitignore` contents.
- Symlinks are stored as symlinks (not followed), so a link pointing outside
  the folder is preserved as-is rather than pulling in its target.
- Sockets, device files, and other irregular entries are skipped.
- Re-running on the same day overwrites the previous archive of the same name.

## Project layout

```
main.go                  CLI entry point: flag parsing + orchestration
internal/
  naming/                Builds the output archive filename
  ignore/                Loads and matches against the root .gitignore
  walker/                Walks the source tree, applying ignore rules
  archive/               Writes the matched entries into a tar.gz
```

Each package is unit-tested against its own public API; `main_test.go`
covers the end-to-end flow.

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

This project was built test-first (TDD): each package's behavior is pinned
down by tests before/alongside its implementation.

## License

[MIT](LICENSE)
