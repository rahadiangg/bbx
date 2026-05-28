package auth

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/fail"
)

func newLogoutCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "logout",
		Short: "Remove a context (or the current one)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdctx.G().Config()
			if err != nil {
				return err
			}
			target := name
			if target == "" {
				target = cfg.CurrentContext
			}
			if target == "" {
				return fail.New("no_context", "no context to remove", fail.ExitUsage)
			}
			if _, ok := cfg.Contexts[target]; !ok {
				return fail.New("not_found", "context "+target+" not found", fail.ExitUsage)
			}
			delete(cfg.Contexts, target)
			if cfg.CurrentContext == target {
				cfg.CurrentContext = ""
				for k := range cfg.Contexts {
					cfg.CurrentContext = k
					break
				}
			}
			if err := cmdctx.G().SaveConfig(); err != nil {
				return fail.Wrap(err, "save_config", fail.ExitGeneric)
			}
			cmdctx.G().Stderr("Removed context %q", target)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "context name to remove (default: current)")
	return c
}
