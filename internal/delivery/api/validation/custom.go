package validation

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Custom validation function for password
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Password requirements:
	// - At least 8 characters
	// - Contains at least one uppercase letter
	// - Contains at least one lowercase letter
	// - Contains at least one number
	// - Contains at least one special character

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)

	return len(password) >= 8 && hasUpper && hasLower && hasNumber && hasSpecial
}

// Custom validation function for no spaces
func validateNoSpaces(fl validator.FieldLevel) bool {
	return !strings.Contains(fl.Field().String(), " ")
}
