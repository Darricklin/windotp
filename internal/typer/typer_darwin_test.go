//go:build darwin

package typer

import (
	"bytes"
	"os/exec"
	"path/filepath"
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
