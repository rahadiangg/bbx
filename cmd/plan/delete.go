package plan

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/fail"
)

func newDeleteCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "delete <project-plan-key>",
		Short: "Delete a plan (irreversible)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fail.New("confirmation_required", "pass --yes to confirm deletion of "+args[0], fail.ExitUsage)
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.DeletePlan(cmd.Context(), args[0]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Deleted plan %s", args[0])
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "confirm deletion (required)")
	return c
}
