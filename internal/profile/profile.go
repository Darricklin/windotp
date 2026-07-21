package profile

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: use 1-64 letters, digits, dots, underscores, or hyphens", name)
	}
	return nil
}

func ValidateMatch(match string) error {
	if strings.TrimSpace(match) != match || match == "" || len(match) > 255 {
		return fmt.Errorf("match must contain 1-255 characters without surrounding whitespace")
	}
	for _, r := range match {
		if unicode.IsControl(r) {
			return fmt.Errorf("match must not contain control characters")
		}
	}
	return nil
}
