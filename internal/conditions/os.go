package conditions

import "runtime"

// MatchesOSCondition checks if the current OS matches any of the provided OS conditions
func MatchesOSCondition(osConditions []string) bool {
	if len(osConditions) == 0 {
		return true // No OS conditions means always match
	}

	currentOS := runtime.GOOS

	for _, os := range osConditions {
		if os == currentOS {
			return true
		}
	}

	return false
}
