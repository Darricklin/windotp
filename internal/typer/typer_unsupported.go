//go:build !darwin

package typer

import (
	"fmt"
	"runtime"
)

func platformType(string, Options) error { return unsupportedError() }
func platformCheck() error               { return unsupportedError() }
func platformChoose([]string, string) (string, error) {
	return "", unsupportedError()
}
func platformContext([]string) (FrontContext, error) { return FrontContext{}, unsupportedError() }

func unsupportedError() error {
	return fmt.Errorf("WindTerm typing is only supported on macOS, not %s", runtime.GOOS)
}
