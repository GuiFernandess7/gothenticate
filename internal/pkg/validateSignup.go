package pkg

import (
	"regexp"
	"unicode"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
var nameRegex = regexp.MustCompile(`^[A-Za-zÀ-ÖØ-öø-ÿ]+(?:['-][A-Za-zÀ-ÖØ-öø-ÿ]+)*(?:\s+[A-Za-zÀ-ÖØ-öø-ÿ]+(?:['-][A-Za-zÀ-ÖØ-öø-ÿ]+)*)+$`)

func IsEmailValid(e string) bool {
	if len(e) < 3 && len(e) > 254 {
		return false
	}
	return emailRegex.MatchString(e)
}

func IsNameValid(name string) bool {
	return nameRegex.MatchString(name)
}

func IsPasswordStrong(password string) (bool, []string) {
	var errors []string

	if len(password) < 8 {
		errors = append(errors, "Password must be at least 8 characters long.")
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		errors = append(errors, "Password must contain at least one uppercase letter.")
	}
	if !hasLower {
		errors = append(errors, "Password must contain at least one lowercase letter.")
	}
	if !hasNumber {
		errors = append(errors, "Password must contain at least one number.")
	}
	if !hasSpecial {
		errors = append(errors, "Password must contain at least one special character.")
	}

	return len(errors) == 0, errors
}