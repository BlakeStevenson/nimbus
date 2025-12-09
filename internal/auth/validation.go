package auth

import (
	"regexp"
	"strings"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)
)

// ValidateEmail checks if an email is valid
func ValidateEmail(email string) error {
	if email == "" || !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateUsername checks if a username is valid
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" || !usernameRegex.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}
