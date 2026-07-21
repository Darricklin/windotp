package config

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestSaveLoadAndResolve(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	cfg := New()
	cfg.Profiles["prod"] = Profile{CreatedAt: time.Unix(123, 0).UTC()}
	cfg.DefaultProfile = "prod"
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", info.Mode().Perm())
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	name, err := got.Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if name != "prod" {
		t.Errorf("Resolve() = %q, want prod", name)
	}
}

func TestPathWithoutHomeEnvironment(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("WINDOTP_CONFIG", "")

	originalLookup := lookupCurrentUser
	lookupCurrentUser = func() (*user.User, error) {
		return &user.User{HomeDir: "/Users/shortcut-user"}, nil
	}
	t.Cleanup(func() { lookupCurrentUser = originalLookup })

	wantBase := filepath.Join("/Users/shortcut-user", ".config")
	if runtime.GOOS == "darwin" {
		wantBase = filepath.Join("/Users/shortcut-user", "Library", "Application Support")
	}
	want := filepath.Join(wantBase, "windotp", "config.json")
	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}
