package config

import (
	"os"
	"path/filepath"
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
