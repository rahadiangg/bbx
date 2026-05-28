package config

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/fail"
)

func newUseContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use-context <name>",
		Short: "Switch the active context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdctx.G().Config()
			if err != nil {
				return err
			}
			if _, ok := cfg.Contexts[args[0]]; !ok {
				return fail.New("not_found", "context "+args[0]+" not found", fail.ExitUsage)
			}
			cfg.CurrentContext = args[0]
			if err := cmdctx.G().SaveConfig(); err != nil {
				return fail.Wrap(err, "save_config", fail.ExitGeneric)
			}
			cmdctx.G().Stderr("Switched to context %q", args[0])
			return nil
		},
	}
}

func newListContextsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "contexts",
		Short: "List configured contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdctx.G().Config()
			if err != nil {
				return err
			}
			type row struct {
				Name     string `json:"name"`
				BaseURL  string `json:"base-url"`
				Current  bool   `json:"current"`
			}
			out := make([]row, 0, len(cfg.Contexts))
			for k, v := range cfg.Contexts {
				out = append(out, row{Name: k, BaseURL: v.BaseURL, Current: k == cfg.CurrentContext})
			}
			return cmdctx.G().Emit(out)
		},
	}
}
