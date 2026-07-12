# bkup

A CLI that backs up a folder into a zip file, the way `git archive` would —
but it doesn't require the folder to be a git repository, and it includes
uncommitted files.

It reads the folder's root `.gitignore`, walks the tree, and zips everything
that isn't excluded (also skipping `.git/` unconditionally).

## Install

Requires Go 1.26+.

```sh
go install .
```

This builds the binary into `$(go env GOPATH)/bin/bkup`. Make sure that
directory is on your `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Usage

```
bkup [-o output-dir] [-v] <folder>
```

| Flag | Default | Description |
|---|---|---|
| `-o`, `--output` | `.` (current directory) | Directory to write the zip into |
| `-v`, `--verbose` | off | List each packed file to stdout |

The output file is named `{folderName}_backup_{YYYYMMDD}.zip`.

### Example

```sh
$ bkup -o ~/backups ~/projects/myapp
backed up 42 files to /Users/you/backups/myapp_backup_20260713.zip
```

Running it with no `-o` writes the zip into the current directory:

```sh
$ cd ~/Desktop
$ bkup myapp
backed up 42 files to myapp_backup_20260713.zip
```

## Behavior notes

- Only the source folder's **root** `.gitignore` is honored — nested
  `.gitignore` files in subdirectories are not read. This is a deliberate
  scope decision, not a limitation of the underlying matcher.
- `.git/` is always excluded, regardless of `.gitignore` contents.
- Re-running on the same day overwrites the previous zip of the same name.

## Project layout

```
main.go                  CLI entry point: flag parsing + orchestration
internal/
  naming/                Builds the output zip filename
  ignore/                Loads and matches against the root .gitignore
  walker/                Walks the source tree, applying ignore rules
  archive/                Writes the matched files into a zip
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
