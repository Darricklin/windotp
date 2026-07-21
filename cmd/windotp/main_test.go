package main

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"

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
