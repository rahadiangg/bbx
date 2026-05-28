package auth

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the authenticated Bamboo user for the current context",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			u, err := cli.WhoAmI(cmd.Context())
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(u)
		},
	}
}
