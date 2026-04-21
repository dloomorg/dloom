# dloom

A lightweight, flexible dotfile manager and system bootstrapper for macOS and Linux.

![Build Status](https://github.com/dloomorg/dloom/actions/workflows/build.yml/badge.svg)
![GitHub Release](https://img.shields.io/github/v/release/dloomorg/dloom)
[![Ubuntu Snap](https://snapcraft.io/dloom/badge.svg)](https://snapcraft.io/dloom)
[![Ubuntu Snap](https://snapcraft.io/dloom/trending.svg?name=0)](https://snapcraft.io/dloom)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

## Overview

**dloom** (pronounced *dee-loom*) is a CLI (command-line interface) tool that _links_ and _unlinks_ configuration files (or _dotfiles_) in a development machine. It manages symlinks between a dotfiles repository (or any source directory) and the machine's target directory (_home_ directory by default; overridable). The tool is inspired from GNU Stow and other dotfile managers, but differs in its approach by creating symlinks for individual files instead of directories. This enables other applications to add files to the same parent directories without them needing to be tracked in the dotfiles repository.

## Features

- **Symlink Management**: Easily create and manage symlinks for your dotfiles.
- **File-Level Symlinks**: Links individual files (not directories), allowing other applications to add files to the same directories without them being tracked in your dotfiles repository (e.g., `~/.ssh/`).
  - This is the main difference from GNU Stow. _It means that the addition of a file to a directory in your dotfiles repository will not automatically create a symlink for it. You will need to run `dloom link` again to create the symlink for the new file._
  - Additionally, links for files can have different names in the target directory. This allows users to have separate dotfiles for different environments (e.g., macOS vs. Linux) without needing to maintain separate branches or repositories, while still having the same name for the symlinked file.
- **Conditional Linking**: Link files only when specific conditions are met (OS, distro, installed tools, tool versions).
- **Customize Setup (Optional)**: Allows customization of how the system is set up using a configuration file. Override settings at the global, package, or file level, including support for regex patterns.
- **Backup System**: Automatically backs up existing files before replacing them.
- **Dry Run Mode**: Preview changes without modifying your system.
- **Cross-Platform**: Works consistently across macOS and Linux.
  - Windows support has not been tested, but contributions are welcome.

## Installation

### Homebrew (macOS and Linux)

```bash
brew tap dloomorg/dloom https://github.com/dloomorg/dloom
brew install dloomorg/dloom/dloom
```

### Arch Linux (AUR)

For Arch Linux users, `dloom` is available in the AUR in two variants. The current AUR packages target `x86_64`:

- `dloom`: builds from source
- `dloom-bin`: installs the pre-built Linux `x86_64` release binary

If you use an AUR helper such as `yay` or `paru`:

```bash
# Build from source
yay -S dloom

# Or install the pre-built binary package
yay -S dloom-bin
```

To install manually from the AUR, clone the package you want and build it with `makepkg`:

```bash
# Build from source
git clone https://aur.archlinux.org/dloom.git
cd dloom
makepkg -si

# Or install the pre-built binary package
git clone https://aur.archlinux.org/dloom-bin.git
cd dloom-bin
makepkg -si
```

### Pre-built Binaries

`dloom` ships as separate binaries for each supported OS and CPU architecture. Download the release asset that matches your platform, extract it, and place the `dloom` binary in your `PATH`.

### From Source

**Requirements:**
- Go 1.26.2 or later

```bash
# Install from source
go install github.com/dloomorg/dloom

# Or clone and build
git clone https://github.com/dloomorg/dloom.git
cd dloom
go build -o bin/dloom
```

## Quick Start

dloom has two main commands: `link` and `unlink`. The `link` command creates symlinks for your dotfiles, while the `unlink` command removes them.

### Linking Dotfiles

To link your dotfiles, run:

```bash
# Link all dotfiles from your vim package
dloom link vim

# Link multiple packages
dloom link vim tmux bash

# Link all packages in the current dotfiles directory
dloom link .

# Link all packages from a specific dotfiles directory
dloom link ~/projects/dotfiles/

# Link with verbose output
dloom -v link vim

# Preview changes without making them
dloom -d link vim
```

### How Symlinks Work

Consider this example dotfiles repository:

```
~/dotfiles/
├── vim/
    ├── .vimrc
    └── .config/
        └── plugins.vim
```

When you run `dloom link vim`, it will create:

```
~/                                                          ~/dotfiles/
  ├── .vimrc ------------------------------------------>      ├── vim/.vimrc
  └── .config/ (regular directory; created if not exists)     └── .config/
      └── plugins.vim --------------------------------->          └── plugins.vim
```

Notice that:
- Only files get symlinked, not directories.
- The directory structure is mirrored in the target; i.e. the home (`~`) directory.
- Files in the same target directories from other sources remain untouched.

Different commands and their effects:

```bash
# Link vim package to home directory
dloom link vim

# Link vim package to a different target directory
dloom -t ~/.config/nvim link vim
# Creates: ~/.config/nvim/.vimrc → ~/dotfiles/vim/.vimrc
#          ~/.config/nvim/.config/plugins.vim → ~/dotfiles/vim/.config/plugins.vim

# Link from a different source directory
dloom -s /path/to/dotfiles link vim
# Uses: /path/to/dotfiles/vim/ as the source

# Dry run to preview changes
dloom -d link vim
# Output:
# Would create a regular directory: /home/user/.config
# Would link: /home/user/.vimrc → /home/user/dotfiles/vim/.vimrc
# Would link: /home/user/.config/plugins.vim → /home/user/dotfiles/vim/.config/plugins.vim
```

### Unlinking Dotfiles
To remove the symlinks created by `dloom`, use the `unlink` command:

```bash
# Unlink all dotfiles from your vim package
dloom unlink vim

# Unlink all packages in the current dotfiles directory
dloom unlink .
```

Unlink will only remove links if they were created by `dloom`, i.e - if the links are pointing to files in the source (usually the dotfiles) directory. Any extra files in the target directory will remain untouched. If `dloom` finds any backups for files that were unlinked, it will restore them. Finally, if the target directory becomes empty after unlinking (and if no backups were found), the directory will be removed. 

### Backup System

`dloom` automatically backs up existing files before replacing them. The backup files are stored in the `backup_dir` specified in the configuration file (default: `~/.dloom/backups`). If a file already exists at the target location, it will be backed up before creating the symlink. During the unlinking process, if a backup exists, it will be restored.

### Dry Run
To preview what would happen without making any changes, use the `-d` or `--dry-run` option:

```bash
dloom -d link vim
```

Sample output:

```
➜  dotfiles git:(main) ✗ dloom -d link dloom zsh alacritty kitty ghostty eza sway waybar
[DRY RUN]: Would link: /home/user_name/.config/dloom/config.yaml -> /home/user_name/dotfiles/dloom/config.yaml
[DRY RUN]: Would link: /home/user_name/.config/zsh/.zshrc-common.zsh -> /home/user_name/dotfiles/zsh/.config/zsh/.zshrc-common.zsh
[DRY RUN]: Would link: /home/user_name/.config/zsh/.zshrc-git.zsh -> /home/user_name/dotfiles/zsh/.config/zsh/.zshrc-git.zsh
[DRY RUN]: Would link: /home/user_name/.config/zsh/.zshrc-nflx.zsh -> /home/user_name/dotfiles/zsh/.config/zsh/.zshrc-nflx.zsh
[DRY RUN]: Would link: /home/user_name/.zshrc -> /home/user_name/dotfiles/zsh/.config/zsh/.zshrc-ubuntu.zsh
[DRY RUN]: Would link: /home/user_name/.config/alacritty/alacritty.toml -> /home/user_name/dotfiles/alacritty/.config/alacritty/alacritty.toml
[DRY RUN]: Would link: /home/user_name/.config/alacritty/catppuccin-mocha.toml -> /home/user_name/dotfiles/alacritty/.config/alacritty/catppuccin-mocha.toml
[DRY RUN]: Would link: /home/user_name/.config/kitty/current-theme.conf -> /home/user_name/dotfiles/kitty/.config/kitty/current-theme.conf
[DRY RUN]: Would link: /home/user_name/.config/kitty/kitty.conf -> /home/user_name/dotfiles/kitty/.config/kitty/kitty.conf
[DRY RUN]: Would link: /home/user_name/.config/ghostty/config -> /home/user_name/dotfiles/ghostty/.config/ghostty/config
[DRY RUN]: Would link: /home/user_name/.config/ghostty/themes/tilix -> /home/user_name/dotfiles/ghostty/.config/ghostty/themes/tilix
[DRY RUN]: Would link: /home/user_name/.config/eza/theme.yml -> /home/user_name/dotfiles/eza/.config/eza/theme.yml
[DRY RUN]: Would link: /home/user_name/.config/sway/config -> /home/user_name/dotfiles/sway/.config/sway/config
[DRY RUN]: Would link: /home/user_name/.config/waybar/configc.json -> /home/user_name/dotfiles/waybar/.config/waybar/configc_sway.json
[DRY RUN]: Would link: /home/user_name/.config/waybar/mocha.css -> /home/user_name/dotfiles/waybar/.config/waybar/mocha.css
[DRY RUN]: Would link: /home/user_name/.config/waybar/start_waybar.sh -> /home/user_name/dotfiles/waybar/.config/waybar/start_waybar_sway.sh
[DRY RUN]: Would link: /home/user_name/.config/waybar/style.css -> /home/user_name/dotfiles/waybar/.config/waybar/style.css
```

This will show what files would be linked or unlinked without actually performing the operation.

## Configuration (Optional)

`dloom` can be customized using a configuration file to override default settings. This allows users to specify different source and target directories, enable verbose output, force overwriting existing files, and set up conditional linking based on various criteria. Some example use cases include:
- If you have different `waybar` config files for `sway` and `hyprland`, you can use `dloom` to link the correct one based on the presence of the executable (`hyprland` or `sway`) on the machine.
- If you have different dotfiles for different operating systems, you can use `dloom` to link the correct one based on the OS. For instance, a different `~/.zshrc` for macOS and Linux.
  - For example, you can have a `.zshrc_linux` and a `~/.zshrc_mac` file in your dotfiles repository. Then, you can use `dloom` to link to the correct one based on the OS. The symlink can be configured to be the standard `~/.zshrc` file, but the actual file in the dotfiles repository is the operating system specific one. This way, you can have different configuration files for different operating systems without needing to maintain separate branches or repositories.
- If you are sharing a dotfiles repository with multiple users, you can use `dloom` to link the correct files based on the user. For instance, a different `~/.zshrc` for different users.

Configuration is done via a YAML file, which allows hierarchical overrides from global, package-specific to individual file-specific settings. By default, `dloom` looks for a `config.yaml` file in the following directories in order of precedence:
1. `./dloom/config.yaml` (in current working directory)
2. `~/.config/dloom/config.yaml` (in user config directory)

You can also specify a custom config file location with the `-c path/to/config.yaml` option. For easiest configuration, create a `dloom/config.yaml` file in the root of your dotfiles repository. You can also use `dloom` itself to link to this file in your repository from `~/.config/dloom/config.yaml` so that you can run `dloom` from anywhere.

### Basic Configuration

```yaml
# Global settings
source_dir: "~/dotfiles"        # Where your dotfiles are stored; default is current directory
target_dir: "~"                 # Where to create symlinks; default is home directory
backup_dir: "~/.dloom/backups"  # Where to back up existing files; default is ~/.dloom/backups
verbose: true                   # Enable detailed output; default is false
force: false                    # Don't overwrite without asking; default is false
dry_run: false                  # Actually make changes; default is false

# Package-specific settings
link_overrides:
  # The package name; this is just a name to group some related files together; 
  # This does not need to be an executable in the system; 
  # dloom expects a directory with the same name in the source_dir, under which the files to be linked will be found
  vim:
    target_dir: "~/.config/nvim"  # Override target for all files under the vim package
    conditions: # Conditions for linking; multiple conditions can be specified and are AND-ed together
      os: # Operating system; multiple OS can be specified; multiple conditions are OR-ed together
        - "linux"
        - "darwin"  # Only link on Linux or macOS
```

### Advanced Configuration

```yaml
link_overrides:
  tmux:
    conditions:
      executable:
        - "tmux"  # Only link if tmux is installed
    
    # File-specific configuration overrides (optional)
    # This overrides package settings and only needed if defaults are not sufficient
    # File name matching attempts to match the exact file name first 
    # before trying regex matching regardless of the declaration order of the overrides
    file_overrides:
      # File with regex pattern matching
      "regex:^tmux.*\.local$":
        conditions: # Multiple conditions specified; will be linked only if all conditions are met
          os:
            - "darwin"  # Only link on macOS
          user:
            - "user_name"  # Only link for user 'user_name'
      
      # Version-specific configurations
      "tmux.new.conf": # File name; must match the exact name in the source directory
        target_name: "tmux.conf"  # Creates the link with a different name
        conditions:
          executable_version:
            "tmux": ">=3.0"  # Only link for tmux 3.0+
```

### Full Configuration

For a more complete example, check the [examples](https://github.com/dloomorg/dloom/tree/main/examples/dloom) directory in the repository. It contains various configurations for different setups.

## Usage

### Linking Dotfiles

```bash
# Basic linking
dloom link <package>...

# Link all packages in the current directory
dloom link .

# Link all packages from a specific directory
dloom link ~/dotfiles/

# Link with options
dloom -v -f link <package>...  # Verbose and force overwrite

# Use -d flag for dry run (preview only)
dloom -d link tmux vim
```

### Unlinking Dotfiles

```bash
# Remove symlinks
dloom unlink <package>...

# Unlink all packages in the current directory
dloom unlink .

# Unlink with options
dloom -d unlink <package>...  # Dry run (preview only)
```

### Command-line Options

| Option | Description |
|--------|-------------|
| `-c, --config` | Path to config file |
| `-f, --force` | Force overwrite existing files |
| `-v, --verbose` | Enable verbose output |
| `-d, -n, --dry-run` | Show what would happen without making changes |
| `-s, --source, --src` | Source directory |
| `-t, --target, --dest` | Target directory |

## Conditional Linking

`dloom` supports conditional linking based on conditions specified in the config file. Conditions can be applied at the package level (affect all files in the package) or at the file level (affect individual files or regex-matched groups).

**Logic rules:**
- Multiple condition types are **AND-ed** — all must pass for the file to be linked.
- Multiple values within a single condition type are **OR-ed** — any one matching is enough.

```yaml
link_overrides:
  mypkg:
    conditions:
      os: [linux, darwin]   # AND-ed with other condition types below
      executable: [git]     # both must be true for the package to link
```

### Condition Reference

#### `os` — Operating System

Links only on the specified operating system(s). Values match Go's `runtime.GOOS`: `linux`, `darwin` (macOS), `windows`.

```yaml
conditions:
  os:
    - linux
    - darwin   # link on Linux or macOS, skip on Windows
```

#### `distro` — Linux Distribution

Links only on the specified Linux distribution(s). Detected from `/etc/os-release`. Common values: `ubuntu`, `debian`, `arch`, `fedora`, `nixos`. On non-Linux systems this condition is ignored (always passes).

```yaml
conditions:
  distro:
    - ubuntu
    - debian
```

#### `executable` — Installed Program

Links only if all listed executables are found in `$PATH`. Useful for linking tool-specific configs only when that tool is installed.

```yaml
conditions:
  executable:
    - tmux    # link only if tmux is installed
```

Multiple executables are AND-ed — all must be present:

```yaml
conditions:
  executable:
    - node
    - npm
```

#### `executable_version` — Program Version

Links only if the specified executables meet the given version constraints. Supports operators `>=`, `>`, `<=`, `<`, `=`.

```yaml
conditions:
  executable_version:
    tmux: ">=3.0"     # tmux 3.0 or newer
    node: ">=18.0.0"
```

Combine with `executable` if the program might not be installed at all:

```yaml
conditions:
  executable:
    - tmux
  executable_version:
    tmux: ">=3.0"
```

#### `user` — System User

Links only when the current user matches. Useful for shared machines or shared dotfiles repos with multiple contributors.

```yaml
conditions:
  user:
    - alice
    - bob
```

#### `hostname` — Machine Hostname

Links only when the system hostname matches. Useful for maintaining one dotfiles repo across multiple machines — work laptop, home desktop, servers — and linking only what belongs on each.

```yaml
conditions:
  hostname:
    - work-macbook
    - work-desktop
```

**Example: work vs. personal machine separation**

```yaml
link_overrides:
  # Work-specific configs — only on work machines
  work:
    conditions:
      hostname:
        - work-macbook
        - work-linux

  # Personal configs — only on personal machines
  personal:
    conditions:
      hostname:
        - home-mac
        - home-desktop

  # SSH config differs per machine
  ssh:
    file_overrides:
      "config.work":
        target_name: "config"
        conditions:
          hostname:
            - work-macbook
      "config.personal":
        target_name: "config"
        conditions:
          hostname:
            - home-mac
```

The hostname value is whatever `hostname` returns on your system (or `os.Hostname()` in Go). Run `hostname` in your terminal to confirm the exact value to use.

## Project Structure

```
dloom/
├── dloom.go            # Command-line interface entry point
├── internal/           # Internal implementation
│   ├── config.go       # Configuration handling
│   ├── link.go         # Link implementation
│   ├── unlink.go       # Unlink implementation
│   ├── resolve.go      # Argument resolution and package enumeration
│   └── setup.go        # System setup implementation
└── examples/           # Sample configurations
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Inspired by [GNU Stow](https://www.gnu.org/software/stow/) and other dotfile managers.
- Built with Go.

---

*dloom - Weave your development environment.*
