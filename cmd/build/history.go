package build

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
)

func newHistoryCmd() *cobra.Command {
	var (
		all   bool
		limit int
		max   int
	)
	c := &cobra.Command{
		Use:   "history <plan-key>",
		Short: "List build history for a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			opts := api.PageOpts{MaxResults: max}
			if all {
				items, err := api.Iterate[api.BuildResult](cmd.Context(), opts, limit,
					func(c2 context.Context, o api.PageOpts) (api.Page[api.BuildResult], error) {
						return cli.BuildHistory(c2, args[0], o)
					})
				if err != nil {
					return err
				}
				return cmdctx.G().Emit(buildResultList(items))
			}
			page, err := cli.BuildHistory(cmd.Context(), args[0], opts)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(buildResultList(page.Results))
		},
	}
	c.Flags().IntVar(&limit, "limit", 0, "max items when --all is set")
	c.Flags().IntVar(&max, "max-results", 25, "page size")
	c.Flags().BoolVar(&all, "all", false, "paginate through all results")
	return c
}
