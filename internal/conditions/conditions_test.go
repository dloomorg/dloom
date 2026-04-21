package conditions

import (
	"os"
	"os/user"
	"runtime"
	"testing"

	"github.com/dloomorg/dloom/internal/logging"
)

// --- hostname ---

func TestMatchesHostnameCondition_Empty(t *testing.T) {
	if !MatchesHostnameCondition(nil) {
		t.Error("nil conditions should always match")
	}
	if !MatchesHostnameCondition([]string{}) {
		t.Error("empty conditions should always match")
	}
}

func TestMatchesHostnameCondition_Match(t *testing.T) {
	h, err := os.Hostname()
	if err != nil {
		t.Skip("cannot determine hostname")
	}
	if !MatchesHostnameCondition([]string{h}) {
		t.Errorf("expected match for current hostname %q", h)
	}
}

func TestMatchesHostnameCondition_NoMatch(t *testing.T) {
	if MatchesHostnameCondition([]string{"nonexistent-hostname-xyz-123"}) {
		t.Error("expected no match for fake hostname")
	}
}

func TestMatchesHostnameCondition_MultipleOneMatches(t *testing.T) {
	h, err := os.Hostname()
	if err != nil {
		t.Skip("cannot determine hostname")
	}
	if !MatchesHostnameCondition([]string{"nonexistent-hostname-xyz-123", h}) {
		t.Errorf("expected match when current hostname %q is in the list", h)
	}
}

// --- os ---

func TestMatchesOSCondition_Empty(t *testing.T) {
	if !MatchesOSCondition(nil) {
		t.Error("nil conditions should always match")
	}
	if !MatchesOSCondition([]string{}) {
		t.Error("empty conditions should always match")
	}
}

func TestMatchesOSCondition_Match(t *testing.T) {
	if !MatchesOSCondition([]string{runtime.GOOS}) {
		t.Errorf("expected match for current OS %q", runtime.GOOS)
	}
}

func TestMatchesOSCondition_NoMatch(t *testing.T) {
	if MatchesOSCondition([]string{"nonexistent-os"}) {
		t.Error("expected no match for fake OS")
	}
}

func TestMatchesOSCondition_MultipleOneMatches(t *testing.T) {
	if !MatchesOSCondition([]string{"nonexistent-os", runtime.GOOS}) {
		t.Errorf("expected match when current OS %q is in the list", runtime.GOOS)
	}
}

// --- user ---

func TestMatchesUserCondition_Empty(t *testing.T) {
	if !MatchesUserCondition(nil) {
		t.Error("nil conditions should always match")
	}
	if !MatchesUserCondition([]string{}) {
		t.Error("empty conditions should always match")
	}
}

func TestMatchesUserCondition_Match(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skip("cannot determine current user")
	}
	if !MatchesUserCondition([]string{u.Username}) {
		t.Errorf("expected match for current user %q", u.Username)
	}
}

func TestMatchesUserCondition_NoMatch(t *testing.T) {
	if MatchesUserCondition([]string{"nonexistent-user-xyz-123"}) {
		t.Error("expected no match for fake user")
	}
}

func TestMatchesUserCondition_MultipleOneMatches(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skip("cannot determine current user")
	}
	if !MatchesUserCondition([]string{"nonexistent-user-xyz-123", u.Username}) {
		t.Errorf("expected match when current user %q is in the list", u.Username)
	}
}

// --- executable ---

func TestMatchesExecutableCondition_Empty(t *testing.T) {
	if !MatchesExecutableCondition(nil) {
		t.Error("nil conditions should always match")
	}
	if !MatchesExecutableCondition([]string{}) {
		t.Error("empty conditions should always match")
	}
}

func TestMatchesExecutableCondition_Found(t *testing.T) {
	if !MatchesExecutableCondition([]string{"go"}) {
		t.Error("expected match for 'go' which must be in PATH to run tests")
	}
}

func TestMatchesExecutableCondition_NotFound(t *testing.T) {
	if MatchesExecutableCondition([]string{"nonexistent-executable-xyz-123"}) {
		t.Error("expected no match for fake executable")
	}
}

func TestMatchesExecutableCondition_MultipleAllFound(t *testing.T) {
	if !MatchesExecutableCondition([]string{"go", "ls"}) {
		t.Error("expected match when all executables are in PATH")
	}
}

func TestMatchesExecutableCondition_MultipleOneMissing(t *testing.T) {
	if MatchesExecutableCondition([]string{"go", "nonexistent-xyz-123"}) {
		t.Error("expected no match when one executable is missing")
	}
}

// --- executable version (pure logic, no process spawning) ---

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2 string
		want   int
	}{
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"1.2.3", "1.2.4", -1},
		{"1.10.0", "1.9.0", 1},  // numeric comparison: 10 > 9
		{"3.0", "2.9", 1},
		{"1.0", "1.0.0", -1},    // fewer parts = smaller
		{"1.0.0", "1.0", 1},     // more parts = larger
	}
	for _, tt := range tests {
		got := compareVersions(tt.v1, tt.v2)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
		}
	}
}

func TestVersionMeetsConstraint(t *testing.T) {
	tests := []struct {
		version, constraint string
		want                bool
	}{
		{"3.0.0", ">=3.0.0", true},
		{"3.1.0", ">=3.0.0", true},
		{"2.9.0", ">=3.0.0", false},
		{"3.0.0", ">3.0.0", false},
		{"3.0.1", ">3.0.0", true},
		{"2.0.0", "<=3.0.0", true},
		{"3.0.0", "<=3.0.0", true},
		{"3.0.1", "<=3.0.0", false},
		{"2.9.0", "<3.0.0", true},
		{"3.0.0", "<3.0.0", false},
		{"3.0.0", "=3.0.0", true},
		{"3.0.1", "=3.0.0", false},
		{"3.0.0", "3.0.0", true}, // no operator = exact match
	}
	for _, tt := range tests {
		got := versionMeetsConstraint(tt.version, tt.constraint)
		if got != tt.want {
			t.Errorf("versionMeetsConstraint(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
		}
	}
}

func TestExtractVersionFromOutput(t *testing.T) {
	tests := []struct {
		output string
		want   string
	}{
		{"git version 2.39.0", "2.39.0"},   // "version X.Y.Z" pattern
		{"v1.21.0", "1.21.0"},              // "vX.Y.Z" pattern
		{"Python 3.11.2", "3.11.2"},        // bare "X.Y.Z" pattern
		{"node v18.12.0\n", "18.12.0"},     // "vX.Y.Z" with trailing newline
		{"OpenSSL 3.0\n", "3.0"},           // "X.Y" at end of line
		{"no version here", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractVersionFromOutput(tt.output)
		if got != tt.want {
			t.Errorf("extractVersionFromOutput(%q) = %q, want %q", tt.output, got, tt.want)
		}
	}
}

func TestMatchesExecutableVersionCondition_Empty(t *testing.T) {
	logger := &logging.Logger{}
	if !MatchesExecutableVersionCondition(nil, logger) {
		t.Error("nil conditions should always match")
	}
	if !MatchesExecutableVersionCondition(map[string]string{}, logger) {
		t.Error("empty conditions should always match")
	}
}

func TestMatchesExecutableVersionCondition_GoVersion(t *testing.T) {
	logger := &logging.Logger{}
	// go must exist (we're running tests with it) and must be >= 1.0
	if !MatchesExecutableVersionCondition(map[string]string{"go": ">=1.0"}, logger) {
		t.Error("expected 'go' to meet >=1.0 constraint")
	}
}

func TestMatchesExecutableVersionCondition_NotFound(t *testing.T) {
	logger := &logging.Logger{}
	if MatchesExecutableVersionCondition(map[string]string{"nonexistent-xyz-123": ">=1.0"}, logger) {
		t.Error("expected no match for non-existent executable")
	}
}

// --- distro ---

func TestParseOSRelease(t *testing.T) {
	tests := []struct {
		content string
		want    string
	}{
		{"NAME=\"Ubuntu\"\nID=ubuntu\nVERSION_ID=\"22.04\"", "ubuntu"},
		{"ID=arch\nNAME=\"Arch Linux\"", "arch"},
		{"ID=\"fedora\"", "fedora"},
		{"NAME=\"Debian GNU/Linux\"", ""},  // no ID= line
		{"", ""},
	}
	for _, tt := range tests {
		got := parseOSRelease(tt.content)
		if got != tt.want {
			t.Errorf("parseOSRelease(%q) = %q, want %q", tt.content, got, tt.want)
		}
	}
}

func TestMatchesDistroCondition_Empty(t *testing.T) {
	if !MatchesDistroCondition(nil) {
		t.Error("nil conditions should always match")
	}
	if !MatchesDistroCondition([]string{}) {
		t.Error("empty conditions should always match")
	}
}

func TestMatchesDistroCondition_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("test only applies to non-Linux systems")
	}
	// On non-Linux, distro conditions are ignored — always pass
	if !MatchesDistroCondition([]string{"ubuntu", "arch"}) {
		t.Error("distro conditions on non-Linux should always match")
	}
}
