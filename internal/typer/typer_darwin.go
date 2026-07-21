//go:build darwin

package typer

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
)

var sixDigits = regexp.MustCompile(`^[0-9]{6}$`)

func platformType(code string, opts Options) error {
	if !sixDigits.MatchString(code) {
		return fmt.Errorf("refusing to type a value that is not exactly six digits")
	}
	guard := `if name of frontProcess is not "WindTerm" then error "WindTerm is not the frontmost application"`
	if opts.AllowAnyApp {
		guard = ""
	}
	enter := ""
	if opts.Enter {
		enter = "key code 36"
	}
	script := fmt.Sprintf(`tell application "System Events"
set frontProcess to first application process whose frontmost is true
%s
keystroke "%s"
%s
end tell
`, guard, code, enter)
	cmd := exec.Command("/usr/bin/osascript")
	cmd.Stdin = bytes.NewBufferString(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("type into WindTerm: %s: %w", bytes.TrimSpace(output), err)
	}
	return nil
}

func platformCheck() error {
	cmd := exec.Command("/usr/bin/osascript")
	cmd.Stdin = bytes.NewBufferString(`tell application "System Events" to get name of first application process whose frontmost is true`)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("macOS Accessibility check failed: %s: %w", bytes.TrimSpace(output), err)
	}
	return nil
}
