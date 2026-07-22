package typer

import (
	"errors"
	"fmt"
	"time"
)

var ErrCanceled = errors.New("profile selection canceled")

type Options struct {
	Enter            bool
	AllowAnyApp      bool
	ActivateWindTerm bool
}

type FrontContext struct {
	Window     string   `json:"window"`
	Candidates []string `json:"candidates"`
}

func Type(code string, opts Options) error {
	return platformType(code, opts)
}

func Check() error {
	return platformCheck()
}

func Choose(profiles []string, defaultProfile string) (string, error) {
	return platformChoose(profiles, defaultProfile)
}

func Context(matches []string) (FrontContext, error) {
	return platformContext(matches)
}

func WaitForPrompt(prompts []string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		visible, err := platformPromptVisible(prompts)
		if err != nil {
			return err
		}
		if visible {
			return nil
		}
		if time.Now().Add(interval).After(deadline) {
			return fmt.Errorf("timed out after %s waiting for WindTerm MFA prompts %q", timeout, prompts)
		}
		time.Sleep(interval)
	}
}
