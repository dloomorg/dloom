package conditions

import (
	"os"
	"sync"
)

var (
	currentHostname string
	hostnameOnce    sync.Once
)

// MatchesHostnameCondition checks if the current hostname matches any of the provided hostname conditions
func MatchesHostnameCondition(hostnameConditions []string) bool {
	if len(hostnameConditions) == 0 {
		return true // No hostname conditions means always match
	}

	current := getHostname()

	for _, h := range hostnameConditions {
		if h == current {
			return true
		}
	}

	return false
}

func getHostname() string {
	hostnameOnce.Do(func() {
		h, err := os.Hostname()
		if err != nil {
			currentHostname = ""
		} else {
			currentHostname = h
		}
	})

	return currentHostname
}
