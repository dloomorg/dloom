package conditions

import "os/exec"

func MatchesExecutableCondition(executableConditions []string) bool {
	if len(executableConditions) == 0 {
		return true // No executable conditions means always match
	}

	for _, execName := range executableConditions {
		// Look for the executable in PATH
		_, err := exec.LookPath(execName)
		if err != nil {
			// Executable not found
			return false
		}
	}

	// All executables were found
	return true
}
