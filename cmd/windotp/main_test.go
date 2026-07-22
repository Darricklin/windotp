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
	cfg.Profiles["jump1"] = config.Profile{CreatedAt: time.Now(), Match: "jump-bj.sensetime.com"}
	cfg.Profiles["jump2"] = config.Profile{CreatedAt: time.Now(), Match: "jump-sh.sensetime.com"}

	got, err := matchProfile(cfg, []string{"WindTerm", "jump-bj.sensetime.com"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "jump1" {
		t.Errorf("matchProfile() = %q, want jump1", got)
	}
}

func TestMatchProfileRejectsMissingAndAmbiguousMatches(t *testing.T) {
	cfg := config.New()
	cfg.Profiles["jump"] = config.Profile{Match: "sensetime.com"}
	cfg.Profiles["jump-bj"] = config.Profile{Match: "jump-bj.sensetime.com"}

	if _, err := matchProfile(cfg, []string{"unrelated.example.com"}); err == nil {
		t.Fatal("missing match unexpectedly succeeded")
	}
	if _, err := matchProfile(cfg, []string{"jump-bj.sensetime.com"}); err == nil {
		t.Fatal("ambiguous match unexpectedly succeeded")
	}
}

func TestMatchesProfile(t *testing.T) {
	tests := []struct {
		name    string
		match   string
		sources []string
		want    bool
	}{
		{name: "selected tab", match: "jump-bj.sensetime.com", sources: []string{"WindTerm", "jump-bj.sensetime.com"}, want: true},
		{name: "case insensitive", match: "JUMP-BJ", sources: []string{"jump-bj.sensetime.com"}, want: true},
		{name: "different tab", match: "jump-bj", sources: []string{"jump-sh.sensetime.com"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesProfile(tt.match, tt.sources); got != tt.want {
				t.Fatalf("matchesProfile(%q, %q) = %v, want %v", tt.match, tt.sources, got, tt.want)
			}
		})
	}
}

func TestTriggerValidatesArguments(t *testing.T) {
	t.Setenv("WINDOTP_CONFIG", t.TempDir()+"/config.json")
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing profile", args: []string{"trigger"}},
		{name: "negative delay", args: []string{"trigger", "--delay=-1ms", "prod"}},
		{name: "invalid minimum validity", args: []string{"trigger", "--min-validity=30s", "prod"}},
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
