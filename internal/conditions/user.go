package conditions

import (
	"os/user"
	"sync"
)

var (
	currentUserName string
	once            sync.Once
)

// MatchesUserCondition checks if the current user matches any of the provided user conditions
func MatchesUserCondition(userConditions []string) bool {
	if len(userConditions) == 0 {
		return true // No user conditions means always match
	}

	var currentUser = getUserName()

	for _, userName := range userConditions {
		if userName == currentUser {
			return true
		}
	}

	return false
}

// GetUserName returns the current user's name
func getUserName() string {
	once.Do(func() {
		usr, err := user.Current()
		if err != nil {
			currentUserName = ""
		} else {
			currentUserName = usr.Username
		}
	})

	return currentUserName
}
