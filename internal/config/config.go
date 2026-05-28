package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the on-disk shape of ~/.config/bbx/config.yaml.
type Config struct {
	CurrentContext string             `yaml:"current-context"`
	Contexts       map[string]Context `yaml:"contexts"`
}

// Context holds the connection settings for a single Bamboo instance.
//
// ServerVersion / ServerBuildNumber / ServerEdition are captured at
// `bbx auth login` time from /rest/api/latest/info. They are NOT used for
// authorization — they exist so future commands can gate behavior on the
// server's major version without an extra round-trip per call. They become
// stale if the Bamboo server is upgraded; `bbx info` refreshes them.
type Context struct {
	BaseURL            string `yaml:"base-url"`
	Token              string `yaml:"token,omitempty"`
	TokenEnv           string `yaml:"token-env,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecure-skip-verify,omitempty"`
	ServerVersion      string `yaml:"server-version,omitempty"`
	ServerBuildNumber  string `yaml:"server-build,omitempty"`
	ServerEdition      string `yaml:"server-edition,omitempty"`
}

// Active returns the currently selected context plus its name.
// Environment variable BBX_TOKEN (or an explicit token-env) overrides
// the stored token if set.
func (c *Config) Active() (string, Context, error) {
	if c == nil || len(c.Contexts) == 0 {
		return "", Context{}, fmt.Errorf("no contexts configured; run `bbx auth login`")
	}
	name := c.CurrentContext
	if name == "" {
		return "", Context{}, fmt.Errorf("no current context selected; run `bbx config use-context <name>`")
	}
	ctx, ok := c.Contexts[name]
	if !ok {
		return "", Context{}, fmt.Errorf("current context %q not found in config", name)
	}
	// env-var resolution
	envKey := ctx.TokenEnv
	if envKey == "" {
		envKey = "BBX_TOKEN"
	}
	if v := os.Getenv(envKey); v != "" {
		ctx.Token = v
	}
	return name, ctx, nil
}

// DefaultPath returns the resolved location of the config file, honoring
// $BBX_CONFIG and $XDG_CONFIG_HOME.
func DefaultPath() (string, error) {
	if p := os.Getenv("BBX_CONFIG"); p != "" {
		return p, nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "bbx", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "bbx", "config.yaml"), nil
}

// Load reads the config file at path. If the file does not exist an empty
// Config is returned (not an error).
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Contexts: map[string]Context{}}, nil
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Contexts == nil {
		cfg.Contexts = map[string]Context{}
	}
	return &cfg, nil
}

// Save writes the config atomically with 0600 perms.
func Save(path string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
