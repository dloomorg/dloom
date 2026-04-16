# CLAUDE.md

## Build & Test

```bash
make build          # Build binary to bin/dloom
make test           # Run all tests (go test -v ./...)
make clean          # Remove bin/
make run            # Build and run
```

## Architecture

Go CLI tool using `cobra` for commands and `gopkg.in/yaml.v3` for config parsing.

### Key directories

- `cmd/` - Cobra command handlers (link, unlink, setup, root)
- `internal/` - Core logic (no external consumers)
  - `config.go` - Config struct, YAML loading, effective config merging (global -> package -> file)
  - `link.go` - Symlink creation via `filepath.Walk`
  - `unlink.go` - Symlink removal with backup restoration
  - `resolve.go` - Argument classification (bare name vs path) and package enumeration
  - `io_utils.go` - `ExpandPath()` for `~` expansion, `copyFile()`
  - `conditions/` - OS, distro, executable, version, user condition matchers
  - `logging/` - Logger with color support

### How linking works

1. CLI args are classified by `resolve.go`: bare names (`vim`) pass through as package names; paths (`.`, `/abs/path`) are resolved to enumerate packages or extract a package name
2. `LinkPackages` iterates package names, calls `linkPackage` for each
3. `linkPackage` gets effective config via `GetEffectiveConfig(pkgName, "")`, checks conditions, then walks `GetSourcePath(pkgName)` creating file-level symlinks in target_dir
4. Config merging: global defaults -> `link_overrides.<pkg>` -> `file_overrides.<file>` (exact match first, then `regex:` patterns)

### Config search order

1. Explicit `-c path`
2. `./dloom/config.yaml` (CWD)
3. `~/.config/dloom/config.yaml`
4. Default config (source_dir=`.`, target_dir=`~`)

## Conventions

- Packages are directories within source_dir; the package name is the directory name
- File-level symlinks only (not directory symlinks) - this is a deliberate design choice vs stow
- `IgnorePackages` default: `.git`, `.idea`, `.gitignore` - uses `strings.HasSuffix` matching
- `*bool` pointer fields in PackageConfig/FileConfig allow distinguishing "not set" from "set to false"
- No test fixtures exist outside `internal/resolve_test.go` - tests use `t.TempDir()`
