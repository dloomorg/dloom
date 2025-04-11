package conditions

import (
	"fmt"
	"github.com/dloomorg/dloom/internal/logging"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// MatchesExecutableVersionCondition checks if executables meet version requirements
func MatchesExecutableVersionCondition(versionConditions map[string]string, logger *logging.Logger) bool {
	if len(versionConditions) == 0 {
		return true // No version conditions means always match
	}

	for execName, versionConstraint := range versionConditions {
		// First check if the executable exists
		execPath, err := exec.LookPath(execName)
		if err != nil {
			// Executable not found
			return false
		}

		// Get the version of the executable
		version, err := getExecutableVersion(execPath, execName)
		if err != nil {
			// Couldn't determine version
			logger.LogWarning("Warning: Could not determine version of %s: %v", execName, err)
			return false
		}

		// Compare the version against the constraint
		if !versionMeetsConstraint(version, versionConstraint) {
			// Version doesn't meet constraint
			return false
		}
	}

	// All version conditions passed
	return true
}

// getExecutableVersion attempts to determine the version of an executable
func getExecutableVersion(execPath, execName string) (string, error) {
	// Common version flags for different programs
	versionFlags := [][]string{
		{"--version"},
		{"-v"},
		{"-V"},
		{"version"},
		{"--ver"},
	}

	// Try each version flag
	for _, flags := range versionFlags {
		cmd := exec.Command(execPath, flags...)
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			// Successfully got output, try to extract version
			version := extractVersionFromOutput(string(output))
			if version != "" {
				return version, nil
			}
		}
	}

	// Special handling for specific executables
	switch execName {
	case "node":
		// Node.js often works with -v
		cmd := exec.Command(execPath, "-v")
		output, err := cmd.CombinedOutput()
		if err == nil {
			return strings.TrimSpace(string(output)), nil
		}
	case "python", "python3":
		// Python has a specific version flag
		cmd := exec.Command(execPath, "--version")
		output, err := cmd.CombinedOutput()
		if err == nil {
			return extractVersionFromOutput(string(output)), nil
		}
	}

	return "", fmt.Errorf("could not determine version")
}

// extractVersionFromOutput tries to find a version string in command output
func extractVersionFromOutput(output string) string {
	// Common version patterns
	patterns := []*regexp.Regexp{
		// Match "version X.Y.Z"
		regexp.MustCompile(`(?i)version\s+(\d+\.\d+\.\d+)`),
		// Match "vX.Y.Z"
		regexp.MustCompile(`v(\d+\.\d+\.\d+)`),
		// Match "X.Y.Z"
		regexp.MustCompile(`(\d+\.\d+\.\d+)`),
		// Match "X.Y"
		regexp.MustCompile(`(\d+\.\d+)\b`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(output)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	return ""
}

// versionMeetsConstraint checks if a version meets a version constraint
func versionMeetsConstraint(version, constraint string) bool {
	// Extract operator and required version
	operator := ""
	requiredVersion := constraint

	// Check for comparison operators
	if strings.HasPrefix(constraint, ">=") {
		operator = ">="
		requiredVersion = strings.TrimPrefix(constraint, ">=")
	} else if strings.HasPrefix(constraint, ">") {
		operator = ">"
		requiredVersion = strings.TrimPrefix(constraint, ">")
	} else if strings.HasPrefix(constraint, "<=") {
		operator = "<="
		requiredVersion = strings.TrimPrefix(constraint, "<=")
	} else if strings.HasPrefix(constraint, "<") {
		operator = "<"
		requiredVersion = strings.TrimPrefix(constraint, "<")
	} else if strings.HasPrefix(constraint, "=") {
		operator = "="
		requiredVersion = strings.TrimPrefix(constraint, "=")
	}

	// Trim whitespace
	requiredVersion = strings.TrimSpace(requiredVersion)
	version = strings.TrimSpace(version)

	// Compare versions
	comparison := compareVersions(version, requiredVersion)

	switch operator {
	case ">=":
		return comparison >= 0
	case ">":
		return comparison > 0
	case "<=":
		return comparison <= 0
	case "<":
		return comparison < 0
	case "=", "":
		return comparison == 0
	default:
		return false
	}
}

// compareVersions compares two version strings
// Returns:
//
//	-1 if v1 < v2
//	 0 if v1 == v2
//	 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// Split versions into components
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each component
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		// If we've run out of components in v1, v1 is less than v2
		if i >= len(parts1) {
			return -1
		}

		// If we've run out of components in v2, v1 is greater than v2
		if i >= len(parts2) {
			return 1
		}

		// Try to convert components to integers
		num1, err1 := strconv.Atoi(parts1[i])
		num2, err2 := strconv.Atoi(parts2[i])

		// If both convert to integers, compare numerically
		if err1 == nil && err2 == nil {
			if num1 < num2 {
				return -1
			} else if num1 > num2 {
				return 1
			}
			// If equal, continue to next component
			continue
		}

		// If not both numbers, compare as strings
		if comp := strings.Compare(parts1[i], parts2[i]); comp != 0 {
			return comp
		}
	}

	// All components are equal
	return 0
}
