package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Darricklin/windotp/internal/config"
	"github.com/Darricklin/windotp/internal/profile"
	"github.com/Darricklin/windotp/internal/store"
	"github.com/Darricklin/windotp/internal/totp"
	"github.com/Darricklin/windotp/internal/typer"
	"golang.org/x/term"
)

var version = "dev"

type app struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	store  store.Store
}

func main() {
	a := app{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr, store: store.New()}
	if err := a.run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "windotp: %v\n", err)
		os.Exit(1)
	}
}

func (a app) run(args []string) error {
	if len(args) == 0 {
		a.usage()
		return nil
	}
	switch args[0] {
	case "add":
		return a.add(args[1:])
	case "list":
		return a.list(args[1:])
	case "default":
		return a.setDefault(args[1:])
	case "bind":
		return a.bind(args[1:])
	case "code":
		return a.code(args[1:])
	case "type":
		return a.typeCode(args[1:])
	case "choose":
		return a.choose(args[1:])
	case "auto":
		return a.auto(args[1:])
	case "trigger":
		return a.trigger(args[1:])
	case "popup":
		return a.popup(args[1:])
	case "remove":
		return a.remove(args[1:])
	case "doctor":
		return a.doctor(args[1:])
	case "version", "--version", "-v":
		fmt.Fprintf(a.stdout, "windotp %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return nil
	case "help", "--help", "-h":
		a.usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q; run windotp help", args[0])
	}
}

func (a app) usage() {
	fmt.Fprint(a.stdout, `WindOTP securely types JumpServer TOTP codes into WindTerm.

Usage:
  windotp add [--stdin] [--default] NAME
  windotp list
  windotp default NAME
  windotp bind NAME WINDTERM_TAB_MATCH
  windotp code [NAME]
  windotp type [--enter=true] [--min-validity=5s] [--delay=0] [NAME]
  windotp choose [--enter=true] [--min-validity=5s]
  windotp auto [--enter=true] [--min-validity=5s]
  windotp trigger [--enter=true] [--min-validity=5s] [--delay=200ms] [--trust-profile] NAME
  windotp popup [--enter=true] [--min-validity=5s] [--timeout=60s] [--interval=200ms] [--delay=100ms] [--trust-profile] NAME
  windotp remove NAME
  windotp doctor
  windotp version

Secrets are stored in macOS Keychain. Profile names are stored in the user config.
`)
}

func (a app) load() (string, config.Config, error) {
	path, err := config.Path()
	if err != nil {
		return "", config.Config{}, err
	}
	cfg, err := config.Load(path)
	return path, cfg, err
}

func (a app) add(args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	fromStdin := fs.Bool("stdin", false, "read the secret or otpauth URL from stdin")
	makeDefault := fs.Bool("default", false, "make this the default profile")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: windotp add [--stdin] [--default] NAME")
	}
	name := fs.Arg(0)
	if err := profile.ValidateName(name); err != nil {
		return err
	}

	secretInput, err := a.readSecret(*fromStdin)
	if err != nil {
		return err
	}
	secret, err := totp.NormalizeSecret(secretInput)
	if err != nil {
		return err
	}
	secretBytes := []byte(secret)
	defer wipe(secretBytes)

	path, cfg, err := a.load()
	if err != nil {
		return err
	}
	_, existed := cfg.Profiles[name]
	if err := a.store.Put(name, secretBytes); err != nil {
		return err
	}
	if !existed {
		cfg.Profiles[name] = config.Profile{CreatedAt: time.Now().UTC()}
	}
	if *makeDefault || len(cfg.Profiles) == 1 {
		cfg.DefaultProfile = name
	}
	if err := config.Save(path, cfg); err != nil {
		if !existed {
			_ = a.store.Delete(name)
		}
		return err
	}
	verb := "added"
	if existed {
		verb = "updated"
	}
	fmt.Fprintf(a.stdout, "%s profile %q\n", verb, name)
	return nil
}

func (a app) readSecret(fromStdin bool) (string, error) {
	if fromStdin {
		data, err := io.ReadAll(io.LimitReader(a.stdin, 16*1024))
		if err != nil {
			return "", fmt.Errorf("read secret: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	file, ok := a.stdin.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", fmt.Errorf("stdin is not a terminal; use --stdin to read the secret explicitly")
	}
	fmt.Fprint(a.stderr, "Base32 secret or otpauth URL: ")
	data, err := term.ReadPassword(int(file.Fd()))
	fmt.Fprintln(a.stderr)
	if err != nil {
		return "", fmt.Errorf("read hidden secret: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func (a app) list(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: windotp list")
	}
	_, cfg, err := a.load()
	if err != nil {
		return err
	}
	for _, name := range cfg.Names() {
		marker := " "
		if name == cfg.DefaultProfile {
			marker = "*"
		}
		fmt.Fprintf(a.stdout, "%s %s\n", marker, name)
	}
	return nil
}

func (a app) setDefault(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: windotp default NAME")
	}
	path, cfg, err := a.load()
	if err != nil {
		return err
	}
	name, err := cfg.Resolve(args[0])
	if err != nil {
		return err
	}
	cfg.DefaultProfile = name
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "default profile is now %q\n", name)
	return nil
}

func (a app) bind(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: windotp bind NAME WINDTERM_TAB_MATCH")
	}
	name, match := args[0], args[1]
	if err := profile.ValidateMatch(match); err != nil {
		return err
	}
	path, cfg, err := a.load()
	if err != nil {
		return err
	}
	if _, err := cfg.Resolve(name); err != nil {
		return err
	}
	entry := cfg.Profiles[name]
	entry.Match = match
	cfg.Profiles[name] = entry
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "bound profile %q to WindTerm tab match %q\n", name, match)
	return nil
}

func (a app) code(args []string) error {
	fs := flag.NewFlagSet("code", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	noNewline := fs.Bool("no-newline", false, "do not print a trailing newline")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("usage: windotp code [--no-newline] [NAME]")
	}
	name := ""
	if fs.NArg() == 1 {
		name = fs.Arg(0)
	}
	name, secret, err := a.resolveSecret(name)
	if err != nil {
		return err
	}
	defer wipe(secret)
	value, err := totp.Code(string(secret), time.Now())
	if err != nil {
		return fmt.Errorf("generate code for %q: %w", name, err)
	}
	fmt.Fprint(a.stdout, value)
	if !*noNewline {
		fmt.Fprintln(a.stdout)
	}
	return nil
}

func (a app) typeCode(args []string) error {
	fs := flag.NewFlagSet("type", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	enter := fs.Bool("enter", true, "press Enter after typing")
	allowAnyApp := fs.Bool("allow-any-app", false, "disable the WindTerm foreground safety check")
	minValidity := fs.Duration("min-validity", 5*time.Second, "wait for a new code when less validity remains")
	delay := fs.Duration("delay", 0, "delay before typing (useful for WindTerm triggers)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("usage: windotp type [OPTIONS] [NAME]")
	}
	if *minValidity < 0 || *minValidity >= totp.Period {
		return fmt.Errorf("min-validity must be between 0 and %s", totp.Period)
	}
	if *delay < 0 {
		return fmt.Errorf("delay must not be negative")
	}
	name := ""
	if fs.NArg() == 1 {
		name = fs.Arg(0)
	}
	return a.typeProfile(name, *minValidity, *delay, typer.Options{Enter: *enter, AllowAnyApp: *allowAnyApp})
}

func (a app) choose(args []string) error {
	fs := flag.NewFlagSet("choose", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	enter := fs.Bool("enter", true, "press Enter after typing")
	minValidity := fs.Duration("min-validity", 5*time.Second, "wait for a new code when less validity remains")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: windotp choose [OPTIONS]")
	}
	if *minValidity < 0 || *minValidity >= totp.Period {
		return fmt.Errorf("min-validity must be between 0 and %s", totp.Period)
	}
	_, cfg, err := a.load()
	if err != nil {
		return err
	}
	selected, err := typer.Choose(cfg.Names(), cfg.DefaultProfile)
	if errors.Is(err, typer.ErrCanceled) {
		return nil
	}
	if err != nil {
		return err
	}
	return a.typeProfile(selected, *minValidity, 0, typer.Options{Enter: *enter, ActivateWindTerm: true})
}

func (a app) auto(args []string) error {
	fs := flag.NewFlagSet("auto", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	enter := fs.Bool("enter", true, "press Enter after typing")
	minValidity := fs.Duration("min-validity", 5*time.Second, "wait for a new code when less validity remains")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: windotp auto [OPTIONS]")
	}
	if *minValidity < 0 || *minValidity >= totp.Period {
		return fmt.Errorf("min-validity must be between 0 and %s", totp.Period)
	}
	_, cfg, err := a.load()
	if err != nil {
		return err
	}
	matches := make([]string, 0, len(cfg.Profiles))
	for _, name := range cfg.Names() {
		match := cfg.Profiles[name].Match
		if match == "" {
			match = name
		}
		matches = append(matches, match)
	}
	context, err := typer.Context(matches)
	if err != nil {
		return err
	}
	sources := append([]string{context.Window}, context.Candidates...)
	name, err := matchProfile(cfg, sources)
	if err != nil {
		return err
	}
	return a.typeProfile(name, *minValidity, 0, typer.Options{Enter: *enter})
}

func (a app) trigger(args []string) error {
	fs := flag.NewFlagSet("trigger", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	enter := fs.Bool("enter", true, "press Enter after typing")
	minValidity := fs.Duration("min-validity", 5*time.Second, "wait for a new code when less validity remains")
	delay := fs.Duration("delay", 200*time.Millisecond, "wait for WindTerm to finish handling the prompt")
	trustProfile := fs.Bool("trust-profile", false, "type without verifying that the active tab matches the profile")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: windotp trigger [OPTIONS] NAME")
	}
	if *minValidity < 0 || *minValidity >= totp.Period {
		return fmt.Errorf("min-validity must be between 0 and %s", totp.Period)
	}
	if *delay < 0 {
		return fmt.Errorf("delay must not be negative")
	}

	_, cfg, err := a.load()
	if err != nil {
		return err
	}
	name, err := cfg.Resolve(fs.Arg(0))
	if err != nil {
		return err
	}

	if *delay > 0 {
		time.Sleep(*delay)
	}
	if !*trustProfile {
		if err := verifyActiveProfile(cfg, name); err != nil {
			return err
		}
	}
	return a.typeProfile(name, *minValidity, 0, typer.Options{Enter: *enter})
}

func (a app) popup(args []string) error {
	fs := flag.NewFlagSet("popup", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	enter := fs.Bool("enter", true, "press Enter after typing")
	minValidity := fs.Duration("min-validity", 5*time.Second, "wait for a new code when less validity remains")
	timeout := fs.Duration("timeout", 60*time.Second, "maximum time to wait for the WindTerm MFA dialog")
	interval := fs.Duration("interval", 200*time.Millisecond, "interval between Accessibility checks")
	delay := fs.Duration("delay", 100*time.Millisecond, "wait after the MFA input appears before typing")
	trustProfile := fs.Bool("trust-profile", false, "type without verifying that the active tab matches the profile")
	prompt := fs.String("prompt", "Please enter 6 digits", "text identifying the WindTerm MFA dialog")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: windotp popup [OPTIONS] NAME")
	}
	if *minValidity < 0 || *minValidity >= totp.Period {
		return fmt.Errorf("min-validity must be between 0 and %s", totp.Period)
	}
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}
	if *interval <= 0 {
		return fmt.Errorf("interval must be greater than zero")
	}
	if *delay < 0 {
		return fmt.Errorf("delay must not be negative")
	}
	if strings.TrimSpace(*prompt) == "" {
		return fmt.Errorf("prompt must not be empty")
	}

	_, cfg, err := a.load()
	if err != nil {
		return err
	}
	name, err := cfg.Resolve(fs.Arg(0))
	if err != nil {
		return err
	}
	if err := typer.WaitForPrompt(*prompt, *timeout, *interval); err != nil {
		return err
	}
	if !*trustProfile {
		if err := verifyActiveProfile(cfg, name); err != nil {
			return err
		}
	}
	return a.typeProfile(name, *minValidity, *delay, typer.Options{Enter: *enter})
}

func verifyActiveProfile(cfg config.Config, name string) error {
	match := cfg.Profiles[name].Match
	if match == "" {
		match = name
	}
	context, err := typer.Context([]string{match})
	if err != nil {
		return err
	}
	sources := append([]string{context.Window}, context.Candidates...)
	if matchesProfile(match, sources) {
		return nil
	}
	if len(nonEmpty(sources)) == 0 {
		return fmt.Errorf("cannot read the active WindTerm tab label; grant Accessibility access to WindTerm for triggers or Automator for shortcuts, then restart that application")
	}
	return fmt.Errorf("trigger for profile %q does not match the active WindTerm tab; detected labels: %q; if WindTerm does not expose tab labels, use --trust-profile only when this session is kept in the foreground", name, nonEmpty(sources))
}

func matchProfile(cfg config.Config, sources []string) (string, error) {
	matched := make([]string, 0, 1)
	for _, name := range cfg.Names() {
		match := cfg.Profiles[name].Match
		if match == "" {
			match = name
		}
		if matchesProfile(match, sources) {
			matched = append(matched, name)
		}
	}
	if len(matched) == 0 {
		labels := nonEmpty(sources)
		if len(labels) == 0 {
			return "", fmt.Errorf("cannot read the active WindTerm tab label; grant Accessibility access to Automator, then restart Automator and WindTerm")
		}
		return "", fmt.Errorf("no profile matches the active WindTerm tab; detected labels: %q", labels)
	}
	if len(matched) > 1 {
		return "", fmt.Errorf("active WindTerm tab matches multiple profiles: %s", strings.Join(matched, ", "))
	}
	return matched[0], nil
}

func matchesProfile(match string, sources []string) bool {
	match = strings.ToLower(match)
	for _, source := range sources {
		if strings.Contains(strings.ToLower(source), match) {
			return true
		}
	}
	return false
}

func (a app) typeProfile(name string, minValidity, delay time.Duration, opts typer.Options) error {
	name, secret, err := a.resolveSecret(name)
	if err != nil {
		return err
	}
	defer wipe(secret)

	if delay > 0 {
		time.Sleep(delay)
	}
	now := time.Now()
	if totp.Remaining(now) < minValidity {
		time.Sleep(totp.Remaining(now) + 50*time.Millisecond)
		now = time.Now()
	}
	value, err := totp.Code(string(secret), now)
	if err != nil {
		return fmt.Errorf("generate code for %q: %w", name, err)
	}
	return typer.Type(value, opts)
}

func (a app) resolveSecret(name string) (string, []byte, error) {
	_, cfg, err := a.load()
	if err != nil {
		return "", nil, err
	}
	name, err = cfg.Resolve(name)
	if err != nil {
		return "", nil, err
	}
	secret, err := a.store.Get(name)
	if err != nil {
		return "", nil, fmt.Errorf("profile %q: %w", name, err)
	}
	return name, secret, nil
}

func (a app) remove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: windotp remove NAME")
	}
	name := args[0]
	path, cfg, err := a.load()
	if err != nil {
		return err
	}
	if _, err := cfg.Resolve(name); err != nil {
		return err
	}
	if err := a.store.Delete(name); err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	delete(cfg.Profiles, name)
	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = ""
		if names := cfg.Names(); len(names) == 1 {
			cfg.DefaultProfile = names[0]
		}
	}
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "removed profile %q\n", name)
	return nil
}

func (a app) doctor(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: windotp doctor")
	}
	fmt.Fprintf(a.stdout, "[ok] platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	path, cfg, err := a.load()
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "[ok] config: %s (%d profiles)\n", path, len(cfg.Profiles))
	if len(cfg.Profiles) > 0 {
		name := cfg.Names()[0]
		secret, err := a.store.Get(name)
		if err != nil {
			return fmt.Errorf("Keychain check for %q failed: %w", name, err)
		}
		wipe(secret)
		fmt.Fprintln(a.stdout, "[ok] macOS Keychain access")
	} else {
		fmt.Fprintln(a.stdout, "[--] macOS Keychain access: add a profile to test")
	}
	if err := typer.Check(); err != nil {
		return fmt.Errorf("Accessibility check failed (grant your terminal access in System Settings > Privacy & Security > Accessibility): %w", err)
	}
	fmt.Fprintln(a.stdout, "[ok] macOS Accessibility automation")
	return nil
}

func wipe(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
