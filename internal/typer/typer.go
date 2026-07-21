package typer

import "errors"

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
