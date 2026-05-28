package project

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <project-key>",
		Short: "Get project metadata by key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			p, err := cli.GetProject(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(p)
		},
	}
}
