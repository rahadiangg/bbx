package plan

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

func newEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <project-plan-key>",
		Short: "Enable a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.EnablePlan(cmd.Context(), args[0]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Enabled plan %s", args[0])
			return nil
		},
	}
}
