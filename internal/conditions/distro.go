package conditions

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// MatchesDistroCondition checks if the current Linux distribution matches any of the provided distro conditions
func MatchesDistroCondition(distroConditions []string) bool {
	if len(distroConditions) == 0 {
		return true // No distro conditions means always match
	}

	// If not on Linux, distro conditions don't apply
	if runtime.GOOS != "linux" {
		return true
	}

	// Try to detect the Linux distribution
	currentDistro := detectLinuxDistribution()
	if currentDistro == "" {
		// Couldn't detect distribution
		return false
	}

	// Check if the current distribution matches any of the conditions
	for _, distro := range distroConditions {
		if strings.EqualFold(distro, currentDistro) {
			return true
		}
	}

	return false
}

// detectLinuxDistribution attempts to determine the current Linux distribution
func detectLinuxDistribution() string {
	// Try reading /etc/os-release first (most modern distros)
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		return parseOSRelease(string(data))
	}

	// Try other common distribution files
	for _, file := range []string{"/etc/lsb-release", "/etc/debian_version", "/etc/fedora-release", "/etc/redhat-release"} {
		if _, err := os.Stat(file); err == nil {
			// Extract distribution name from filename
			base := filepath.Base(file)
			switch {
			case strings.Contains(base, "debian"):
				return "debian"
			case strings.Contains(base, "ubuntu") || strings.Contains(base, "lsb"):
				return "ubuntu"
			case strings.Contains(base, "fedora"):
				return "fedora"
			case strings.Contains(base, "redhat"):
				return "rhel"
			}
		}
	}

	// Check for Arch Linux
	if _, err := os.Stat("/etc/arch-release"); err == nil {
		return "arch"
	}

	// Couldn't determine the distribution
	return ""
}

// parseOSRelease extracts the distribution ID from /etc/os-release content
func parseOSRelease(content string) string {
	// Look for ID= line
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			// Remove quotes if present
			id = strings.Trim(id, "\"'")
			return id
		}
	}
	return ""
}
