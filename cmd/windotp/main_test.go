package main

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Darricklin/windotp/internal/config"
	"github.com/Darricklin/windotp/internal/store"
)

type memoryStore map[string][]byte

func (m memoryStore) Put(name string, secret []byte) error {
	m[name] = append([]byte(nil), secret...)
	return nil
}

func (m memoryStore) Get(name string) ([]byte, error) {
	secret, ok := m[name]
	if !ok {
		return nil, store.ErrNotFound
	}
	return append([]byte(nil), secret...), nil
}

func (m memoryStore) Delete(name string) error {
	if _, ok := m[name]; !ok {
		return store.ErrNotFound
	}
	delete(m, name)
	return nil
}

func TestProfileLifecycle(t *testing.T) {
	t.Setenv("WINDOTP_CONFIG", t.TempDir()+"/config.json")
	secrets := memoryStore{}
	var stdout, stderr bytes.Buffer
	a := app{
		stdin:  strings.NewReader("JBSWY3DPEHPK3PXP\n"),
		stdout: &stdout,
		stderr: &stderr,
		store:  secrets,
	}
	if err := a.run([]string{"add", "--stdin", "--default", "prod"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `added profile "prod"`) {
		t.Fatalf("unexpected add output: %q", stdout.String())
	}

	stdout.Reset()
	if err := a.run([]string{"list"}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "* prod\n" {
		t.Fatalf("list output = %q", stdout.String())
	}

	stdout.Reset()
	if err := a.run([]string{"code", "prod"}); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9]{6}\n$`).MatchString(stdout.String()) {
		t.Fatalf("code output = %q", stdout.String())
	}

	stdout.Reset()
	if err := a.run([]string{"remove", "prod"}); err != nil {
		t.Fatal(err)
	}
	if _, err := secrets.Get("prod"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("removed secret still exists: %v", err)
	}
}

func TestAddRejectsSecretArgument(t *testing.T) {
	t.Setenv("WINDOTP_CONFIG", t.TempDir()+"/config.json")
	a := app{
		stdin:  strings.NewReader("JBSWY3DPEHPK3PXP\n"),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		store:  memoryStore{},
	}
	err := a.run([]string{"add", "--secret", "JBSWY3DPEHPK3PXP", "prod"})
	if err == nil {
		t.Fatal("add unexpectedly accepted --secret")
	}
}

func TestMatchProfile(t *testing.T) {
	cfg := config.New()
	cfg.Profiles["jump1"] = config.Profile{CreatedAt: time.Now(), Match: "jump-tap1"}
	cfg.Profiles["jump2"] = config.Profile{CreatedAt: time.Now(), Match: "jump-tap2"}

	got, err := matchProfile(cfg, []string{"WindTerm", "jump-tap1"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "jump1" {
		t.Errorf("matchProfile() = %q, want jump1", got)
	}
}

func TestMatchProfileRejectsMissingAndAmbiguousMatches(t *testing.T) {
	cfg := config.New()
	cfg.Profiles["jump"] = config.Profile{Match: "jump-tap"}
	cfg.Profiles["jump1"] = config.Profile{Match: "jump-tap1"}

	if _, err := matchProfile(cfg, []string{"unrelated.example.com"}); err == nil {
		t.Fatal("missing match unexpectedly succeeded")
	}
	if _, err := matchProfile(cfg, []string{"jump-tap1"}); err == nil {
		t.Fatal("ambiguous match unexpectedly succeeded")
	}
	if _, err := matchProfile(cfg, []string{"", ""}); err == nil || !strings.Contains(err.Error(), "cannot read the active WindTerm tab label") {
		t.Fatalf("empty labels returned unexpected error: %v", err)
	}
}

func TestMatchesProfile(t *testing.T) {
	tests := []struct {
		name    string
		match   string
		sources []string
		want    bool
	}{
		{name: "selected tab", match: "jump-tap1", sources: []string{"WindTerm", "jump-tap1"}, want: true},
		{name: "case insensitive", match: "JUMP-TAP1", sources: []string{"jump-tap1"}, want: true},
		{name: "different tab", match: "jump-tap1", sources: []string{"jump-tap2"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesProfile(tt.match, tt.sources); got != tt.want {
				t.Fatalf("matchesProfile(%q, %q) = %v, want %v", tt.match, tt.sources, got, tt.want)
			}
		})
	}
}

func TestHelpOmitsRemovedTriggerCommand(t *testing.T) {
	var stdout bytes.Buffer
	a := app{stdin: strings.NewReader(""), stdout: &stdout, stderr: &bytes.Buffer{}, store: memoryStore{}}
	if err := a.run([]string{"help"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "windotp trigger") {
		t.Fatalf("help still contains removed trigger command: %q", stdout.String())
	}
	if err := a.run([]string{"trigger", "prod"}); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("removed trigger returned unexpected error: %v", err)
	}
}

func TestPopupValidatesArguments(t *testing.T) {
	t.Setenv("WINDOTP_CONFIG", t.TempDir()+"/config.json")
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing profile", args: []string{"popup"}},
		{name: "negative timeout", args: []string{"popup", "--timeout=-1s", "prod"}},
		{name: "zero interval", args: []string{"popup", "--interval=0", "prod"}},
		{name: "negative delay", args: []string{"popup", "--delay=-1ms", "prod"}},
		{name: "empty prompt", args: []string{"popup", "--prompt=", "prod"}},
		{name: "invalid minimum validity", args: []string{"popup", "--min-validity=30s", "prod"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := app{stdin: strings.NewReader(""), stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}, store: memoryStore{}}
			if err := a.run(tt.args); err == nil {
				t.Fatalf("run(%q) unexpectedly succeeded", tt.args)
			}
		})
	}
}

func TestPopupPromptCandidates(t *testing.T) {
	got := popupPromptCandidates(defaultPopupPrompt)
	want := []string{"Please enter 6 digits", "Please Enter MFA Code"}
	if len(got) != len(want) {
		t.Fatalf("popupPromptCandidates(default) = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("popupPromptCandidates(default) = %q, want %q", got, want)
		}
	}

	custom := popupPromptCandidates("OTP Code")
	if len(custom) != 1 || custom[0] != "OTP Code" {
		t.Fatalf("popupPromptCandidates(custom) = %q, want [OTP Code]", custom)
	}
}
