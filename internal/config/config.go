package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

const currentVersion = 1

var lookupCurrentUser = user.Current

type Profile struct {
	CreatedAt time.Time `json:"created_at"`
}

type Config struct {
	Version        int                `json:"version"`
	DefaultProfile string             `json:"default_profile,omitempty"`
	Profiles       map[string]Profile `json:"profiles"`
}

func New() Config {
	return Config{Version: currentVersion, Profiles: make(map[string]Profile)}
}

func Path() (string, error) {
	if path := os.Getenv("WINDOTP_CONFIG"); path != "" {
		return path, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		current, lookupErr := lookupCurrentUser()
		if lookupErr != nil {
			return "", fmt.Errorf("find user config directory: %w; look up current user: %v", err, lookupErr)
		}
		if current.HomeDir == "" {
			return "", fmt.Errorf("find user config directory: current user has no home directory")
		}
		base = filepath.Join(current.HomeDir, ".config")
		if runtime.GOOS == "darwin" {
			base = filepath.Join(current.HomeDir, "Library", "Application Support")
		}
	}
	return filepath.Join(base, "windotp", "config.json"), nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Version != currentVersion {
		return Config{}, fmt.Errorf("unsupported config version %d", cfg.Version)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func (c Config) Names() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c Config) Resolve(name string) (string, error) {
	if name != "" {
		if _, ok := c.Profiles[name]; !ok {
			return "", fmt.Errorf("profile %q does not exist", name)
		}
		return name, nil
	}
	if c.DefaultProfile != "" {
		if _, ok := c.Profiles[c.DefaultProfile]; ok {
			return c.DefaultProfile, nil
		}
		return "", fmt.Errorf("default profile %q does not exist in config", c.DefaultProfile)
	}
	if len(c.Profiles) == 1 {
		for only := range c.Profiles {
			return only, nil
		}
	}
	if len(c.Profiles) == 0 {
		return "", fmt.Errorf("no profiles configured; run windotp add NAME")
	}
	return "", fmt.Errorf("multiple profiles configured; specify NAME or run windotp default NAME")
}
