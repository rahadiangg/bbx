package build

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <build-result-key>",
		Short: "Get a single build result (e.g. PROJ-PLAN-42)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			br, err := cli.GetBuild(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(br)
		},
	}
}
