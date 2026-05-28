package build

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
)

func newLatestCmd() *cobra.Command {
	var max int
	c := &cobra.Command{
		Use:   "latest",
		Short: "List the latest build across visible plans",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			page, err := cli.LatestBuilds(cmd.Context(), api.PageOpts{MaxResults: max})
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(buildResultList(page.Results))
		},
	}
	c.Flags().IntVar(&max, "max-results", 25, "page size")
	return c
}
