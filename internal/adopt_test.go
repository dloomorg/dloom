package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dloomorg/dloom/internal/logging"
)

func TestAdoptPackage_File(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	homeDir := filepath.Join(tmpDir, "home")

	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(homeDir, ".zshrc")
	contents := []byte("export ZDOTDIR=$HOME/.config/zsh\n")
	if err := os.WriteFile(targetPath, contents, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir:      repoDir,
		TargetDir:      homeDir,
		IgnorePackages: []string{".git", ".idea", ".gitignore"},
	}
	logger := &logging.Logger{UseColors: false}

	err := AdoptPackage(AdoptOptions{
		Config:  cfg,
		Package: "zsh",
		Targets: []string{targetPath},
	}, logger)
	if err != nil {
		t.Fatal(err)
	}

	sourcePath := filepath.Join(repoDir, "zsh", ".zshrc")
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(sourceData) != string(contents) {
		t.Fatalf("source contents = %q, want %q", sourceData, contents)
	}

	targetInfo, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if targetInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", targetPath)
	}

	linkDest, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(linkDest) != filepath.Clean(sourcePath) {
		t.Fatalf("symlink destination = %s, want %s", linkDest, sourcePath)
	}
}

func TestAdoptPackage_DirectoryWithPackageTargetDir(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	homeDir := filepath.Join(tmpDir, "home")
	targetDir := filepath.Join(homeDir, ".config", "ghostty")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(targetDir, "config")
	if err := os.WriteFile(targetPath, []byte("font-size = 13\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir:      repoDir,
		TargetDir:      homeDir,
		IgnorePackages: []string{".git", ".idea", ".gitignore"},
		Packages: map[string]*PackageConfig{
			"ghostty": {
				TargetDir: targetDir,
			},
		},
	}
	logger := &logging.Logger{UseColors: false}

	err := AdoptPackage(AdoptOptions{
		Config:  cfg,
		Package: "ghostty",
		Targets: []string{targetDir},
	}, logger)
	if err != nil {
		t.Fatal(err)
	}

	sourcePath := filepath.Join(repoDir, "ghostty", "config")
	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatal(err)
	}

	linkDest, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(linkDest) != filepath.Clean(sourcePath) {
		t.Fatalf("symlink destination = %s, want %s", linkDest, sourcePath)
	}
}

func TestAdoptPackage_ReusesExistingIdenticalSource(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	homeDir := filepath.Join(tmpDir, "home")

	if err := os.MkdirAll(filepath.Join(repoDir, "zsh"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	sourcePath := filepath.Join(repoDir, "zsh", ".zshrc")
	targetPath := filepath.Join(homeDir, ".zshrc")
	contents := []byte("export ZDOTDIR=$HOME/.config/zsh\n")

	if err := os.WriteFile(sourcePath, contents, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, contents, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir:      repoDir,
		TargetDir:      homeDir,
		IgnorePackages: []string{".git", ".idea", ".gitignore"},
	}
	logger := &logging.Logger{UseColors: false}

	err := AdoptPackage(AdoptOptions{
		Config:  cfg,
		Package: "zsh",
		Targets: []string{targetPath},
	}, logger)
	if err != nil {
		t.Fatal(err)
	}

	linkDest, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(linkDest) != filepath.Clean(sourcePath) {
		t.Fatalf("symlink destination = %s, want %s", linkDest, sourcePath)
	}
}

func TestAdoptPackage_SkipsForeignSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	homeDir := filepath.Join(tmpDir, "home")
	targetDir := filepath.Join(homeDir, ".config", "hypr")
	foreignDir := filepath.Join(tmpDir, "foreign")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(foreignDir, 0755); err != nil {
		t.Fatal(err)
	}

	regularTarget := filepath.Join(targetDir, "hyprland.conf")
	if err := os.WriteFile(regularTarget, []byte("source=~/.config/hypr/colors.conf\n"), 0644); err != nil {
		t.Fatal(err)
	}

	foreignFile := filepath.Join(foreignDir, "wallpaper.conf")
	if err := os.WriteFile(foreignFile, []byte("preexisting symlink target\n"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkTarget := filepath.Join(targetDir, "wallpaper.conf")
	if err := os.Symlink(foreignFile, symlinkTarget); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir:      repoDir,
		TargetDir:      homeDir,
		IgnorePackages: []string{".git", ".idea", ".gitignore"},
	}
	logger := &logging.Logger{UseColors: false}

	err := AdoptPackage(AdoptOptions{
		Config:  cfg,
		Package: "hypr",
		Targets: []string{targetDir},
	}, logger)
	if err != nil {
		t.Fatal(err)
	}

	adoptedSource := filepath.Join(repoDir, "hypr", ".config", "hypr", "hyprland.conf")
	if _, err := os.Stat(adoptedSource); err != nil {
		t.Fatal(err)
	}

	linkDest, err := os.Readlink(regularTarget)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(linkDest) != filepath.Clean(adoptedSource) {
		t.Fatalf("symlink destination = %s, want %s", linkDest, adoptedSource)
	}

	foreignLinkDest, err := os.Readlink(symlinkTarget)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(foreignLinkDest) != filepath.Clean(foreignFile) {
		t.Fatalf("foreign symlink destination = %s, want %s", foreignLinkDest, foreignFile)
	}

	skippedSource := filepath.Join(repoDir, "hypr", ".config", "hypr", "wallpaper.conf")
	if _, err := os.Stat(skippedSource); !os.IsNotExist(err) {
		t.Fatalf("expected skipped symlink source to not exist, got err=%v", err)
	}
}

func TestAdoptPackage_RejectsRegexOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	homeDir := filepath.Join(tmpDir, "home")
	targetDir := filepath.Join(homeDir, ".config", "waybar")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(targetDir, "config.json")
	if err := os.WriteFile(targetPath, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir: repoDir,
		TargetDir: homeDir,
		Packages: map[string]*PackageConfig{
			"waybar": {
				Files: map[string]*FileConfig{
					"regex:.*\\.json$": {
						Verbose: boolPtr(true),
					},
				},
			},
		},
	}
	logger := &logging.Logger{UseColors: false}

	err := AdoptPackage(AdoptOptions{
		Config:  cfg,
		Package: "waybar",
		Targets: []string{targetDir},
	}, logger)
	if err == nil {
		t.Fatal("expected adopt to reject regex file_overrides")
	}
	if !strings.Contains(err.Error(), "regex file_overrides") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdoptPackage_RejectsTargetNameOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	homeDir := filepath.Join(tmpDir, "home")
	targetPath := filepath.Join(homeDir, ".zshrc")

	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("export EDITOR=nvim\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir: repoDir,
		TargetDir: homeDir,
		Packages: map[string]*PackageConfig{
			"zsh": {
				Files: map[string]*FileConfig{
					".zshrc_linux": {
						TargetName: ".zshrc",
					},
				},
			},
		},
	}
	logger := &logging.Logger{UseColors: false}

	err := AdoptPackage(AdoptOptions{
		Config:  cfg,
		Package: "zsh",
		Targets: []string{targetPath},
	}, logger)
	if err == nil {
		t.Fatal("expected adopt to reject target_name overrides")
	}
	if !strings.Contains(err.Error(), "target_name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdoptPackage_RejectsForce(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	homeDir := filepath.Join(tmpDir, "home")
	targetPath := filepath.Join(homeDir, ".zshrc")

	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("export EDITOR=nvim\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir: repoDir,
		TargetDir: homeDir,
		Force:     true,
	}
	logger := &logging.Logger{UseColors: false}

	err := AdoptPackage(AdoptOptions{
		Config:  cfg,
		Package: "zsh",
		Targets: []string{targetPath},
	}, logger)
	if err == nil {
		t.Fatal("expected adopt to reject --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdoptPackage_InteractiveDecline(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	homeDir := filepath.Join(tmpDir, "home")

	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(homeDir, ".zshrc")
	if err := os.WriteFile(targetPath, []byte("export EDITOR=nvim\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir:      repoDir,
		TargetDir:      homeDir,
		IgnorePackages: []string{".git", ".idea", ".gitignore"},
	}
	logger := &logging.Logger{UseColors: false}

	restoreStdin := withTestStdin(t, "n\n")
	defer restoreStdin()

	err := AdoptPackage(AdoptOptions{
		Config:      cfg,
		Package:     "zsh",
		Targets:     []string{targetPath},
		Interactive: true,
	}, logger)
	if err != nil {
		t.Fatal(err)
	}

	targetInfo, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if targetInfo.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected %s to remain a regular file", targetPath)
	}

	sourcePath := filepath.Join(repoDir, "zsh", ".zshrc")
	if _, err := os.Stat(sourcePath); !os.IsNotExist(err) {
		t.Fatalf("expected source file to not exist, got err=%v", err)
	}
}

func TestExecuteAdoptCandidate_InteractiveDryRunDecline(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	homeDir := filepath.Join(tmpDir, "home")

	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(homeDir, ".zshrc")
	if err := os.WriteFile(targetPath, []byte("export EDITOR=nvim\n"), 0644); err != nil {
		t.Fatal(err)
	}

	candidate := adoptCandidate{
		relPath:    ".zshrc",
		sourcePath: filepath.Join(repoDir, "zsh", ".zshrc"),
		targetPath: targetPath,
	}
	if err := planAdoptCandidate(&candidate); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SourceDir:      repoDir,
		TargetDir:      homeDir,
		DryRun:         true,
		IgnorePackages: []string{".git", ".idea", ".gitignore"},
	}
	logger := &logging.Logger{UseColors: false}

	restoreStdin := withTestStdin(t, "n\n")
	defer restoreStdin()

	adopted, err := executeAdoptCandidate(candidate, "zsh", cfg, logger, true)
	if err != nil {
		t.Fatal(err)
	}
	if adopted {
		t.Fatal("expected dry-run interactive decline to skip adoption")
	}

	targetInfo, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if targetInfo.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected %s to remain a regular file", targetPath)
	}

	if _, err := os.Stat(candidate.sourcePath); !os.IsNotExist(err) {
		t.Fatalf("expected source file to not exist, got err=%v", err)
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func withTestStdin(t *testing.T, input string) func() {
	t.Helper()

	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := writer.WriteString(input); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	os.Stdin = reader

	return func() {
		os.Stdin = oldStdin
		_ = reader.Close()
	}
}
