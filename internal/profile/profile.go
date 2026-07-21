package profile

import (
	"fmt"
	"regexp"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: use 1-64 letters, digits, dots, underscores, or hyphens", name)
	}
	return nil
}
