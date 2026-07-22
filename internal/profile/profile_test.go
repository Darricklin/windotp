package profile

import "testing"

func TestValidateMatch(t *testing.T) {
	for _, match := range []string{"jump-tap1", "jump1", "root@jump.example.com"} {
		if err := ValidateMatch(match); err != nil {
			t.Errorf("ValidateMatch(%q): %v", match, err)
		}
	}
	for _, match := range []string{"", " jump1", "jump1 ", "jump1\nother"} {
		if err := ValidateMatch(match); err == nil {
			t.Errorf("ValidateMatch(%q) unexpectedly succeeded", match)
		}
	}
}
