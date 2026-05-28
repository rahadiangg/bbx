package config

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/config"
	"github.com/rahadiangg/bbx/internal/fail"
)

func newSetCmd() *cobra.Command {
	var contextName string
	c := &cobra.Command{
		Use:   "set <key>=<value> [<key>=<value> ...]",
		Short: "Set fields on a context",
		Args:  cobra.MinimumNArgs(1),
		Long: `Set fields on a named context. Without --context, the current context is used.

Supported keys: base-url, token, token-env, insecure-skip-verify`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdctx.G().Config()
			if err != nil {
				return err
			}
			name := contextName
			if name == "" {
				name = cfg.CurrentContext
			}
			if name == "" {
				return fail.New("no_context", "no context selected", fail.ExitUsage)
			}
			ctx := cfg.Contexts[name]
			for _, kv := range args {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return fail.New("invalid_arg", "expected key=value, got "+kv, fail.ExitUsage)
				}
				switch parts[0] {
				case "base-url":
					ctx.BaseURL = strings.TrimRight(parts[1], "/")
				case "token":
					ctx.Token = parts[1]
				case "token-env":
					ctx.TokenEnv = parts[1]
				case "insecure-skip-verify":
					ctx.InsecureSkipVerify = parts[1] == "true" || parts[1] == "1"
				default:
					return fail.New("unknown_key", "unknown config key: "+parts[0], fail.ExitUsage)
				}
			}
			if cfg.Contexts == nil {
				cfg.Contexts = map[string]config.Context{}
			}
			cfg.Contexts[name] = ctx
			if err := cmdctx.G().SaveConfig(); err != nil {
				return fail.Wrap(err, "save_config", fail.ExitGeneric)
			}
			cmdctx.G().Stderr("Updated context %q", name)
			return nil
		},
	}
	c.Flags().StringVar(&contextName, "context", "", "context to modify (default: current)")
	return c
}
