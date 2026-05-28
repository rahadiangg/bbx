package plan

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

func newDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <project-plan-key>",
		Short: "Disable a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.DisablePlan(cmd.Context(), args[0]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Disabled plan %s", args[0])
			return nil
		},
	}
}
