// Package cmdctx wires global state (config, client, output format) into Cobra commands.
package cmdctx

import (
	"context"
	"fmt"
	"os"

	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/config"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

// Globals are populated by the root command's PersistentPreRun and consumed
// by leaf commands via Get().
type Globals struct {
	ConfigPath  string
	ContextName string // override of current-context, may be empty
	FormatFlag  string // raw -o flag value
	Format      output.Format
	Verbose     int

	// Lazily computed
	cfg    *config.Config
	client *api.Client
}

var g Globals

// Set replaces the active globals (called from root.go).
func Set(v Globals) { g = v }

// G returns a pointer to the active globals (mutable).
func G() *Globals { return &g }

// Config returns the loaded config, loading it on first access.
func (g *Globals) Config() (*config.Config, error) {
	if g.cfg != nil {
		return g.cfg, nil
	}
	path := g.ConfigPath
	if path == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return nil, err
		}
		path = p
		g.ConfigPath = p
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	g.cfg = cfg
	return cfg, nil
}

// SaveConfig persists the current in-memory config back to disk.
func (g *Globals) SaveConfig() error {
	if g.cfg == nil {
		return fmt.Errorf("no config loaded")
	}
	return config.Save(g.ConfigPath, g.cfg)
}

// Client returns a Bamboo API client built from the active context.
func (g *Globals) Client(ctx context.Context) (*api.Client, error) {
	if g.client != nil {
		return g.client, nil
	}
	cfg, err := g.Config()
	if err != nil {
		return nil, err
	}
	// Apply -context override
	if g.ContextName != "" {
		cfg.CurrentContext = g.ContextName
	}
	name, c, err := cfg.Active()
	if err != nil {
		return nil, err
	}
	_ = name
	cli, err := api.New(api.Options{
		BaseURL:            c.BaseURL,
		Token:              c.Token,
		InsecureSkipVerify: c.InsecureSkipVerify,
	})
	if err != nil {
		return nil, err
	}
	g.client = cli
	return cli, nil
}

// Emit prints v with the active output format. On error it returns a wrapped fail.Error.
func (g *Globals) Emit(v any) error {
	if err := output.Print(g.Format, v); err != nil {
		return fail.Wrap(err, "output_error", fail.ExitGeneric)
	}
	return nil
}

// Stderr writes a human-readable message to stderr (suppressed when stdout is JSON
// and the message would be redundant).
func (g *Globals) Stderr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
