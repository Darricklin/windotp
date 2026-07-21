package typer

type Options struct {
	Enter       bool
	AllowAnyApp bool
}

func Type(code string, opts Options) error {
	return platformType(code, opts)
}

func Check() error {
	return platformCheck()
}
