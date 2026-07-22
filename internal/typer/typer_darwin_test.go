//go:build darwin

package typer

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrontContextScriptCompiles(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "front-context.scpt")
	cmd := exec.Command("/usr/bin/osacompile", "-l", "JavaScript", "-o", outputPath)
	cmd.Stdin = bytes.NewBufferString("const profileMatches = [];\n" + frontContextScript)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile front context script: %s: %v", bytes.TrimSpace(output), err)
	}
}

func TestPromptVisibleScriptCompiles(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "prompt-visible.scpt")
	cmd := exec.Command("/usr/bin/osacompile", "-l", "JavaScript", "-o", outputPath)
	cmd.Stdin = bytes.NewBufferString("const expectedPrompt = \"Please enter 6 digits\";\n" + promptVisibleScript)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile prompt visibility script: %s: %v", bytes.TrimSpace(output), err)
	}
}

func TestPromptVisibleScriptClearsRememberStep(t *testing.T) {
	for _, marker := range []string{"remember this step", "记住这一步", "AXPress", "rememberCleared"} {
		if !strings.Contains(promptVisibleScript, marker) {
			t.Fatalf("prompt script does not contain %q", marker)
		}
	}
}
