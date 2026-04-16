# Path-Aware Argument Resolution — Review Document

## The Problem

`dloom link` treated all arguments as bare package names (subdirectory names within `source_dir`). This broke in two scenarios:

1. **`dloom link .`** — `GetSourcePath(".")` resolved to `source_dir` itself, walking the entire dotfiles root as a single flat package. All `link_overrides` configs were ignored because the package name was `.`, not actual package names like `vim` or `zsh`.

2. **`dloom link ~/projects/dotfiles/`** — The full path was concatenated as `filepath.Join(source_dir, "~/projects/dotfiles/")`, producing a nonsensical path that didn't exist.

## Design Decision

After researching how GNU Stow handles this (stow also uses packages-as-directories, requires explicit package names, has no flat mode), we decided:

- **Packages are the only unit of linking** — same mental model as stow
- **`dloom link .` means "link all packages"** — enumerate subdirectories in source_dir
- **Root-level files are skipped** with an info message — users should organize into package directories
- **No `--flat` or `-p` flag needed** — keeps the interface simple
- **Absolute paths** (`/abs/path`, `~/path`) are treated as external dotfiles roots and enumerated
- **Relative paths** (`./vim`, `vim/`) within source_dir extract the package name

## Plan

Introduce an argument resolution layer (`internal/resolve.go`) between the CLI commands and the internal link/unlink logic. Arguments are classified as either **bare names** or **paths**, resolved into a clean list of package names, then passed to the existing `LinkPackages`/`UnlinkPackages` functions unchanged.

### Resolution rules

| Argument type | Example | Resolution |
|---|---|---|
| Bare name | `vim`, `zsh` | Pass through as package name (unchanged behavior) |
| `.` | `dloom link .` | Resolves to source_dir → enumerate all package directories |
| Relative path | `./vim`, `vim/` | Child of source_dir → extract relative name as package |
| Absolute/home path | `/abs/path`, `~/dotfiles/` | External dotfiles root → load config from there, enumerate |
| Non-existent path | `/no/such/dir` | Error |

## Implementation Tasks

| # | Task | Status | Details |
|---|------|--------|---------|
| 1 | Create `internal/resolve.go` | DONE | `ClassifyArg()`, `ResolveArgs()`, `EnumeratePackages()`, `isSubdirOf()`, `isAbsoluteArg()`, `shouldIgnoreName()`. Handles symlink resolution (macOS `/var` → `/private/var`). |
| 2 | Create `internal/resolve_test.go` | DONE | 16 tests: `TestClassifyArg` (12 cases), `TestEnumeratePackages` (3 variants), `TestResolveArgs_*` (10 scenarios including dot, subdir, external with/without config, dedup, trailing slash, conflicts, not-a-directory), `Test_isSubdirOf` (5 cases). |
| 3 | Modify `cmd/link.go` and `cmd/unlink.go` | DONE | Both now route args through `ResolveArgs()` before calling `LinkPackages`/`UnlinkPackages`. Added `Long` descriptions to cobra commands documenting the new path support. |
| 4 | Update `README.md` | DONE | Added `dloom link .`, `dloom link ~/dotfiles/`, `dloom unlink .` examples to Quick Start, Usage, and Project Structure sections. |
| 5 | Create `CLAUDE.md` | DONE | Project guide with build commands, architecture overview, config search order, and conventions. |
| 6 | Build and test | DONE | Clean build, all 16 tests pass, manual dry-run verification of all scenarios. |

## Files Changed

### New files
- **`internal/resolve.go`** — Core argument resolution logic (165 lines)
- **`internal/resolve_test.go`** — Comprehensive test suite (370 lines)
- **`CLAUDE.md`** — Project guide for future sessions

### Modified files
- **`cmd/link.go`** — Added `ResolveArgs` call, updated command description
- **`cmd/unlink.go`** — Same changes as link.go
- **`README.md`** — Documented new `link .` and `unlink .` behavior

### Unchanged files
- `internal/config.go` — No changes needed; `dloom` dir stays as a valid linkable package
- `cmd/root.go` — No changes needed; CLI flags already merged into config by `PersistentPreRunE`
- `internal/link.go`, `internal/unlink.go` — Untouched; resolution happens before they're called

## Implementation Note: macOS Symlink Resolution

During testing, discovered that macOS resolves `/var` → `/private/var` differently depending on whether you use `filepath.Abs` (preserves symlinks) vs `os.Getwd` (resolves them). This caused path comparison failures in tests. Fixed by using `filepath.EvalSymlinks` on both the resolved argument path and source_dir before comparing.

## Verification Results

```
$ cd /tmp/test-dotfiles && dloom -d link .
[INFO]: Skipping root-level file (not in a package): .gitignore
[INFO]: Skipping root-level file (not in a package): README.md
[DRY RUN]: Would link: ~/.gitconfig -> /tmp/test-dotfiles/git/.gitconfig
[DRY RUN]: Would link: ~/.vimrc -> /tmp/test-dotfiles/vim/.vimrc
[DRY RUN]: Would link: ~/.zshrc -> /tmp/test-dotfiles/zsh/.zshrc

$ cd /tmp && dloom -d link /tmp/test-dotfiles/
# Same output — correctly enumerates packages from absolute path

$ cd /tmp/test-dotfiles && dloom -d link vim
[DRY RUN]: Would link: ~/.vimrc -> /tmp/test-dotfiles/vim/.vimrc
# Bare name — unchanged behavior

$ cd /tmp/test-dotfiles && dloom -d unlink .
[INFO]: Skipping root-level file (not in a package): .gitignore
[INFO]: Skipping root-level file (not in a package): README.md
# Unlink works symmetrically
```
