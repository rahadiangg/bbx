package plan

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

type planList []api.Plan

func (p planList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "KEY", "NAME", "PROJECT", "ENABLED")
	for _, pl := range p {
		t.AppendRow([]any{pl.Key, pl.Name, pl.ProjectName, pl.Enabled})
	}
	t.Render()
	return nil
}

func newListCmd() *cobra.Command {
	var (
		limit  int
		max    int
		expand string
		all    bool
	)
	c := &cobra.Command{
		Use:   "list",
		Short: "List plans",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			opts := api.PageOpts{MaxResults: max, Expand: expand}
			if all {
				items, err := api.Iterate[api.Plan](cmd.Context(), opts, limit, cli.ListPlans)
				if err != nil {
					return err
				}
				return cmdctx.G().Emit(planList(items))
			}
			page, err := cli.ListPlans(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(planList(page.Results))
		},
	}
	c.Flags().IntVar(&limit, "limit", 0, "max items when --all is set (0 = unlimited)")
	c.Flags().IntVar(&max, "max-results", 25, "page size")
	c.Flags().StringVar(&expand, "expand", "plans", "expansion hints")
	c.Flags().BoolVar(&all, "all", false, "paginate through all results")
	return c
}
