// Package config handles persistent local storage of mcsmcli credentials.
// The config file is located at ~/.config/mcsmcli/config.json with 0600 permissions.
// Multiple profiles (multi-panel) are supported; the environment variables
// MCSM_URL / MCSM_APIKEY / MCSM_DAEMON can override values.
//
// Config parsing is backed by github.com/spf13/viper.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Profile stores credentials for a single panel.
type Profile struct {
	URL    string `json:"url" mapstructure:"url"`                 // panel URL, e.g. https://panel.example.com
	APIKey string `json:"apikey" mapstructure:"apikey"`           // user API key
	Daemon string `json:"daemon,omitempty" mapstructure:"daemon"` // default daemonId (optional)
}

// Config is the top-level config file structure.
type Config struct {
	Current  string              `json:"current" mapstructure:"current"`
	Profiles map[string]*Profile `json:"profiles" mapstructure:"profiles"`
}

// DefaultProfileName is used when no profile is specified.
const DefaultProfileName = "default"

// vp is the package-level viper instance used for all config operations.
var vp = viper.New()

func init() {
	vp.SetConfigName("config")
	vp.SetConfigType("json")

	// Bind environment variables for credential overrides.
	// Priority: flag > env > config file.
	vp.BindEnv("override.url", "MCSM_URL")
	vp.BindEnv("override.apikey", "MCSM_APIKEY")
	vp.BindEnv("override.daemon", "MCSM_DAEMON")
}

// Init completes viper setup once the home directory is known.
// It is safe to call multiple times.
func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	vp.AddConfigPath(filepath.Join(home, ".config", "mcsmcli"))

	// Respect MCSM_CONFIG override for the config file path.
	if p := os.Getenv("MCSM_CONFIG"); p != "" {
		vp.SetConfigFile(p)
	}
	return nil
}

// Path returns the config file path.
func Path() (string, error) {
	if p := os.Getenv("MCSM_CONFIG"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "mcsmcli", "config.json"), nil
}

// Load reads the config file via viper. Returns an empty config if the file
// does not exist.
func Load() (*Config, error) {
	cfg := &Config{Current: DefaultProfileName, Profiles: map[string]*Profile{}}

	if err := vp.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		// When using SetConfigFile (e.g. via MCSM_CONFIG), viper may return
		// an *fs.PathError wrapping os.ErrNotExist instead of ConfigFileNotFoundError.
		if errors.As(err, &notFound) || errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if err := vp.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", vp.ConfigFileUsed(), err)
	}

	// Post-unmarshal defaults.
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	if cfg.Current == "" {
		cfg.Current = DefaultProfileName
	}
	return cfg, nil
}

// Save writes the config to disk using viper. Directories are 0700, file is
// 0600 for credential safety.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Push the in-memory config into viper's key space.
	vp.Set("current", c.Current)
	vp.Set("profiles", c.Profiles)

	// WriteConfigAs uses 0644; tighten to 0600 for credentials.
	if err := vp.WriteConfigAs(path); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("set config file permissions: %w", err)
	}
	return nil
}

// Resolve determines the effective Profile by priority:
// command-line override > environment variable > config file.
// name defaults to Current when empty.
func (c *Config) Resolve(name, urlOverride, keyOverride string) (*Profile, string, error) {
	if name == "" {
		name = c.Current
	}
	p := c.Profiles[name]
	if p == nil {
		p = &Profile{}
	}

	eff := &Profile{URL: p.URL, APIKey: p.APIKey, Daemon: p.Daemon}

	// Environment variable overrides (via viper).
	if v := vp.GetString("override.url"); v != "" {
		eff.URL = v
	}
	if v := vp.GetString("override.apikey"); v != "" {
		eff.APIKey = v
	}
	if v := vp.GetString("override.daemon"); v != "" {
		eff.Daemon = v
	}

	// Command-line flag overrides (highest priority).
	if urlOverride != "" {
		eff.URL = urlOverride
	}
	if keyOverride != "" {
		eff.APIKey = keyOverride
	}

	eff.URL = strings.TrimRight(eff.URL, "/")
	if eff.URL == "" || eff.APIKey == "" {
		return nil, name, fmt.Errorf("profile %q is missing panel URL or API key; run mcsmcli login first", name)
	}
	return eff, name, nil
}
