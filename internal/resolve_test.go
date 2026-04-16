package internal

import (
	"github.com/dloomorg/dloom/internal/logging"
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyArg(t *testing.T) {
	tests := []struct {
		arg  string
		want ArgKind
	}{
		{"vim", ArgBare},
		{"zsh", ArgBare},
		{"my-package", ArgBare},
		{".", ArgPath},
		{"..", ArgPath},
		{"./vim", ArgPath},
		{"../dotfiles", ArgPath},
		{"/absolute/path", ArgPath},
		{"~/dotfiles", ArgPath},
		{"~/dotfiles/vim", ArgPath},
		{"vim/", ArgPath},
		{"sub/dir", ArgPath},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			got := ClassifyArg(tt.arg)
			if got != tt.want {
				t.Errorf("ClassifyArg(%q) = %d, want %d", tt.arg, got, tt.want)
			}
		})
	}
}

func TestEnumeratePackages(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create package directories
	for _, dir := range []string{"vim", "zsh", "git", ".git", ".idea"} {
		if err := os.Mkdir(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Create root-level files
	for _, f := range []string{".zshrc", "README.md", ".gitignore"} {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	logger := &logging.Logger{UseColors: false}
	ignorePackages := []string{".git", ".idea", ".gitignore"}

	packages, err := EnumeratePackages(tmpDir, ignorePackages, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Should return sorted package names, excluding ignored dirs and files
	expected := []string{"git", "vim", "zsh"}
	if len(packages) != len(expected) {
		t.Fatalf("got %d packages %v, want %d packages %v", len(packages), packages, len(expected), expected)
	}
	for i, pkg := range packages {
		if pkg != expected[i] {
			t.Errorf("packages[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

func TestEnumeratePackages_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &logging.Logger{UseColors: false}

	packages, err := EnumeratePackages(tmpDir, nil, logger)
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 0 {
		t.Errorf("expected empty packages, got %v", packages)
	}
}

func TestEnumeratePackages_OnlyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// Only root-level files, no package dirs
	for _, f := range []string{".zshrc", ".gitconfig"} {
		os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644)
	}

	logger := &logging.Logger{UseColors: false}
	packages, err := EnumeratePackages(tmpDir, nil, logger)
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 0 {
		t.Errorf("expected no packages (only root files), got %v", packages)
	}
}

func TestResolveArgs_BareName(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{SourceDir: tmpDir}
	logger := &logging.Logger{UseColors: false}

	packages, effectiveCfg, err := ResolveArgs([]string{"vim", "zsh"}, cfg, logger)
	if err != nil {
		t.Fatal(err)
	}
	if effectiveCfg != cfg {
		t.Error("expected config to be unchanged for bare names")
	}
	expected := []string{"vim", "zsh"}
	if len(packages) != len(expected) {
		t.Fatalf("got %v, want %v", packages, expected)
	}
	for i, pkg := range packages {
		if pkg != expected[i] {
			t.Errorf("packages[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

func TestResolveArgs_Dot(t *testing.T) {
	tmpDir := t.TempDir()
	// Create package dirs
	for _, dir := range []string{"vim", "zsh", ".git"} {
		os.Mkdir(filepath.Join(tmpDir, dir), 0755)
	}
	// Create a root file (should be skipped)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("test"), 0644)

	// source_dir must be absolute (as done in cmd/root.go)
	cfg := &Config{
		SourceDir:      tmpDir,
		IgnorePackages: []string{".git"},
	}
	logger := &logging.Logger{UseColors: false}

	// Change to tmpDir so "." resolves to source_dir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	packages, _, err := ResolveArgs([]string{"."}, cfg, logger)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"vim", "zsh"}
	if len(packages) != len(expected) {
		t.Fatalf("got %v, want %v", packages, expected)
	}
	for i, pkg := range packages {
		if pkg != expected[i] {
			t.Errorf("packages[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

func TestResolveArgs_SubdirOfSourceDir(t *testing.T) {
	tmpDir := t.TempDir()
	vimDir := filepath.Join(tmpDir, "vim")
	os.Mkdir(vimDir, 0755)

	cfg := &Config{SourceDir: tmpDir}
	logger := &logging.Logger{UseColors: false}

	// Use relative path — cd into sourceDir and use ./vim
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	packages, _, err := ResolveArgs([]string{"./vim"}, cfg, logger)
	if err != nil {
		t.Fatal(err)
	}

	if len(packages) != 1 || packages[0] != "vim" {
		t.Errorf("got %v, want [vim]", packages)
	}
}

func TestResolveArgs_AbsolutePathEnumerates(t *testing.T) {
	// Absolute paths should always be treated as dotfiles roots,
	// even if they're technically children of source_dir.
	tmpDir := t.TempDir()
	childDir := filepath.Join(tmpDir, "my-dotfiles")
	os.Mkdir(childDir, 0755)
	os.Mkdir(filepath.Join(childDir, "vim"), 0755)
	os.Mkdir(filepath.Join(childDir, "zsh"), 0755)

	cfg := &Config{
		SourceDir:      tmpDir,
		IgnorePackages: []string{".git"},
	}
	logger := &logging.Logger{UseColors: false}

	packages, effectiveCfg, err := ResolveArgs([]string{childDir}, cfg, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Should enumerate packages from the child dir, not extract "my-dotfiles" as a package
	if effectiveCfg.SourceDir != childDir {
		t.Errorf("expected SourceDir %s, got %s", childDir, effectiveCfg.SourceDir)
	}
	expected := []string{"vim", "zsh"}
	if len(packages) != len(expected) {
		t.Fatalf("got %v, want %v", packages, expected)
	}
	for i, pkg := range packages {
		if pkg != expected[i] {
			t.Errorf("packages[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

func TestResolveArgs_ExternalDirWithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// Create external dotfiles dir with config
	externalDir := filepath.Join(tmpDir, "external-dotfiles")
	os.MkdirAll(filepath.Join(externalDir, "dloom"), 0755)
	os.WriteFile(filepath.Join(externalDir, "dloom", "config.yaml"), []byte("verbose: true\n"), 0644)
	// Create package dirs in external
	os.Mkdir(filepath.Join(externalDir, "tmux"), 0755)
	os.Mkdir(filepath.Join(externalDir, "bash"), 0755)

	cfg := &Config{
		SourceDir:      filepath.Join(tmpDir, "original"),
		IgnorePackages: []string{".git"},
	}
	logger := &logging.Logger{UseColors: false}

	packages, effectiveCfg, err := ResolveArgs([]string{externalDir}, cfg, logger)
	if err != nil {
		t.Fatal(err)
	}

	if effectiveCfg.SourceDir != externalDir {
		t.Errorf("expected SourceDir to be %s, got %s", externalDir, effectiveCfg.SourceDir)
	}

	expected := []string{"bash", "dloom", "tmux"}
	if len(packages) != len(expected) {
		t.Fatalf("got %v, want %v", packages, expected)
	}
	for i, pkg := range packages {
		if pkg != expected[i] {
			t.Errorf("packages[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

func TestResolveArgs_ExternalDirWithoutConfig(t *testing.T) {
	tmpDir := t.TempDir()
	externalDir := filepath.Join(tmpDir, "simple-dotfiles")
	os.Mkdir(externalDir, 0755)
	os.Mkdir(filepath.Join(externalDir, "vim"), 0755)

	cfg := &Config{
		SourceDir:      filepath.Join(tmpDir, "original"),
		IgnorePackages: []string{".git"},
	}
	logger := &logging.Logger{UseColors: false}

	packages, effectiveCfg, err := ResolveArgs([]string{externalDir}, cfg, logger)
	if err != nil {
		t.Fatal(err)
	}

	if effectiveCfg.SourceDir != externalDir {
		t.Errorf("expected SourceDir to be %s, got %s", externalDir, effectiveCfg.SourceDir)
	}
	if len(packages) != 1 || packages[0] != "vim" {
		t.Errorf("got %v, want [vim]", packages)
	}
}

func TestResolveArgs_NonexistentPath(t *testing.T) {
	cfg := &Config{SourceDir: "/tmp/nonexistent-source"}
	logger := &logging.Logger{UseColors: false}

	_, _, err := ResolveArgs([]string{"/tmp/nonexistent-path-xyz"}, cfg, logger)
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestResolveArgs_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()
	for _, dir := range []string{"vim", "zsh"} {
		os.Mkdir(filepath.Join(tmpDir, dir), 0755)
	}

	cfg := &Config{
		SourceDir:      tmpDir,
		IgnorePackages: []string{".git"},
	}
	logger := &logging.Logger{UseColors: false}

	// Change to tmpDir so "." resolves to source_dir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// "." expands to [vim, zsh], then "vim" is also passed — should be deduplicated
	packages, _, err := ResolveArgs([]string{".", "vim"}, cfg, logger)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"vim", "zsh"}
	if len(packages) != len(expected) {
		t.Fatalf("got %v, want %v", packages, expected)
	}
}

func TestResolveArgs_TrailingSlash(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "vim"), 0755)

	cfg := &Config{SourceDir: tmpDir}
	logger := &logging.Logger{UseColors: false}

	// cd into sourceDir and use relative path with trailing slash
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	packages, _, err := ResolveArgs([]string{"./vim/"}, cfg, logger)
	if err != nil {
		t.Fatal(err)
	}

	if len(packages) != 1 || packages[0] != "vim" {
		t.Errorf("got %v, want [vim]", packages)
	}
}

func TestResolveArgs_ConflictingOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dotfiles1")
	dir2 := filepath.Join(tmpDir, "dotfiles2")
	os.Mkdir(dir1, 0755)
	os.Mkdir(dir2, 0755)

	cfg := &Config{SourceDir: filepath.Join(tmpDir, "original")}
	logger := &logging.Logger{UseColors: false}

	_, _, err := ResolveArgs([]string{dir1, dir2}, cfg, logger)
	if err == nil {
		t.Error("expected error for conflicting source directories")
	}
}

func TestResolveArgs_NotADirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "somefile")
	os.WriteFile(filePath, []byte("test"), 0644)

	cfg := &Config{SourceDir: tmpDir}
	logger := &logging.Logger{UseColors: false}

	_, _, err := ResolveArgs([]string{filePath}, cfg, logger)
	if err == nil {
		t.Error("expected error for non-directory path")
	}
}

func Test_isSubdirOf(t *testing.T) {
	tests := []struct {
		child  string
		parent string
		want   bool
	}{
		{"/home/user/dotfiles/vim", "/home/user/dotfiles", true},
		{"/home/user/dotfiles/vim/colors", "/home/user/dotfiles", true},
		{"/home/user/dotfiles", "/home/user/dotfiles", false},
		{"/home/user/other", "/home/user/dotfiles", false},
		{"/home/user/dotfiles-extra", "/home/user/dotfiles", false},
	}

	for _, tt := range tests {
		t.Run(tt.child+"_in_"+tt.parent, func(t *testing.T) {
			got := isSubdirOf(tt.child, tt.parent)
			if got != tt.want {
				t.Errorf("isSubdirOf(%q, %q) = %v, want %v", tt.child, tt.parent, got, tt.want)
			}
		})
	}
}
