//go:build darwin

package typer

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
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
	activate := ""
	if opts.ActivateWindTerm {
		activate = `set windTermProcess to first application process whose name is "WindTerm"
set frontmost of windTermProcess to true
delay 0.2`
	}
	enter := ""
	if opts.Enter {
		enter = "key code 36"
	}
	script := fmt.Sprintf(`tell application "System Events"
%s
set frontProcess to first application process whose frontmost is true
%s
keystroke "%s"
%s
end tell
`, activate, guard, code, enter)
	cmd := exec.Command("/usr/bin/osascript")
	cmd.Stdin = bytes.NewBufferString(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("type into WindTerm: %s: %w", bytes.TrimSpace(output), err)
	}
	return nil
}

func platformChoose(profiles []string, defaultProfile string) (string, error) {
	if len(profiles) == 0 {
		return "", fmt.Errorf("no profiles configured")
	}
	quoted := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		quoted = append(quoted, `"`+profile+`"`)
	}
	defaultClause := ""
	if defaultProfile != "" {
		defaultClause = ` default items {"` + defaultProfile + `"}`
	}
	script := fmt.Sprintf(`tell application "System Events"
set frontProcess to first application process whose frontmost is true
if name of frontProcess is not "WindTerm" then error "WindTerm is not the frontmost application"
end tell
set picked to choose from list {%s} with title "WindOTP" with prompt "Select a JumpServer"%s
if picked is false then return ""
return item 1 of picked
`, strings.Join(quoted, ", "), defaultClause)
	cmd := exec.Command("/usr/bin/osascript")
	cmd.Stdin = bytes.NewBufferString(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("choose JumpServer profile: %s: %w", bytes.TrimSpace(output), err)
	}
	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", ErrCanceled
	}
	return selected, nil
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
