package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	if cfg == nil || cfg.Contexts == nil {
		t.Fatalf("expected non-nil cfg with non-nil Contexts map, got %+v", cfg)
	}
	if len(cfg.Contexts) != 0 || cfg.CurrentContext != "" {
		t.Fatalf("expected empty cfg, got %+v", cfg)
	}
}

func TestLoadParseError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("contexts: [not a map"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatalf("expected parse error, got nil")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.yaml")

	in := &Config{
		CurrentContext: "prod",
		Contexts: map[string]Context{
			"prod": {BaseURL: "https://bamboo.prod", Token: "secret", InsecureSkipVerify: true},
			"dev":  {BaseURL: "https://bamboo.dev", TokenEnv: "BBX_DEV_TOKEN"},
		},
	}
	if err := Save(path, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// File must be 0600
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 perms, got %o", st.Mode().Perm())
	}

	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.CurrentContext != "prod" {
		t.Errorf("CurrentContext = %q, want prod", out.CurrentContext)
	}
	if got := out.Contexts["prod"]; got != in.Contexts["prod"] {
		t.Errorf("prod context = %+v, want %+v", got, in.Contexts["prod"])
	}
	if got := out.Contexts["dev"]; got != in.Contexts["dev"] {
		t.Errorf("dev context = %+v, want %+v", got, in.Contexts["dev"])
	}
}

func TestSaveNilConfigRejected(t *testing.T) {
	t.Parallel()
	if err := Save(filepath.Join(t.TempDir(), "x.yaml"), nil); err == nil {
		t.Fatalf("expected error for nil config")
	}
}

func TestActive(t *testing.T) {
	t.Run("no contexts", func(t *testing.T) {
		c := &Config{}
		if _, _, err := c.Active(); err == nil {
			t.Fatalf("expected error")
		}
	})
	t.Run("missing current-context", func(t *testing.T) {
		c := &Config{Contexts: map[string]Context{"a": {BaseURL: "u"}}}
		if _, _, err := c.Active(); err == nil || !strings.Contains(err.Error(), "no current context") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("current-context not found", func(t *testing.T) {
		c := &Config{CurrentContext: "missing", Contexts: map[string]Context{"a": {BaseURL: "u"}}}
		if _, _, err := c.Active(); err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("env override via BBX_TOKEN", func(t *testing.T) {
		t.Setenv("BBX_TOKEN", "from-env")
		c := &Config{
			CurrentContext: "prod",
			Contexts:       map[string]Context{"prod": {BaseURL: "u", Token: "stored"}},
		}
		name, ctx, err := c.Active()
		if err != nil {
			t.Fatalf("Active: %v", err)
		}
		if name != "prod" {
			t.Errorf("name = %q, want prod", name)
		}
		if ctx.Token != "from-env" {
			t.Errorf("Token = %q, want from-env", ctx.Token)
		}
	})
	t.Run("env override via custom token-env", func(t *testing.T) {
		t.Setenv("BBX_TOKEN", "")
		t.Setenv("MY_CUSTOM_TOKEN", "custom-val")
		c := &Config{
			CurrentContext: "p",
			Contexts:       map[string]Context{"p": {BaseURL: "u", Token: "stored", TokenEnv: "MY_CUSTOM_TOKEN"}},
		}
		_, ctx, err := c.Active()
		if err != nil {
			t.Fatalf("Active: %v", err)
		}
		if ctx.Token != "custom-val" {
			t.Errorf("Token = %q, want custom-val", ctx.Token)
		}
	})
	t.Run("no env, falls back to stored token", func(t *testing.T) {
		t.Setenv("BBX_TOKEN", "")
		c := &Config{
			CurrentContext: "p",
			Contexts:       map[string]Context{"p": {BaseURL: "u", Token: "stored"}},
		}
		_, ctx, err := c.Active()
		if err != nil {
			t.Fatalf("Active: %v", err)
		}
		if ctx.Token != "stored" {
			t.Errorf("Token = %q, want stored", ctx.Token)
		}
	})
}

func TestDefaultPath(t *testing.T) {
	t.Run("BBX_CONFIG wins", func(t *testing.T) {
		t.Setenv("BBX_CONFIG", "/tmp/custom.yaml")
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		if got != "/tmp/custom.yaml" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("BBX_CONFIG", "")
		t.Setenv("XDG_CONFIG_HOME", "/x")
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		if got != "/x/bbx/config.yaml" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("home fallback", func(t *testing.T) {
		t.Setenv("BBX_CONFIG", "")
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/h")
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		if got != "/h/.config/bbx/config.yaml" {
			t.Fatalf("got %q", got)
		}
	})
}
