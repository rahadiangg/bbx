package build

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <build-result-key>",
		Short: "Show the live status of a build (finished/progress)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			s, err := cli.GetBuildStatus(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(s)
		},
	}
}
