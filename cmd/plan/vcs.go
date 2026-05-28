package plan

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

type vcsBranchList []api.VCSBranch

func (v vcsBranchList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "NAME")
	for _, b := range v {
		t.AppendRow([]any{b.Name})
	}
	t.Render()
	return nil
}

// newVCSCmd implements `bbx plan vcs-branches <key>` — lists branches in the
// underlying repository (NOT the plan branches; those have their own command).
func newVCSCmd() *cobra.Command {
	var (
		all   bool
		limit int
		max   int
	)
	c := &cobra.Command{
		Use:   "vcs-branches <plan-key>",
		Short: "List branches present in the plan's underlying repository",
		Long: `List branches that exist in the plan's underlying VCS repository.

This is distinct from 'bbx plan branch list', which lists Bamboo *plan
branches* (each tracked Bamboo branch becomes its own plan). vcs-branches
returns the raw branch names from the repository itself.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			opts := api.PageOpts{MaxResults: max}
			if all {
				items, err := api.Iterate(cmd.Context(), opts, limit,
					func(c2 context.Context, o api.PageOpts) (api.Page[api.VCSBranch], error) {
						return cli.ListPlanVCSBranches(c2, args[0], o)
					})
				if err != nil {
					return err
				}
				return cmdctx.G().Emit(vcsBranchList(items))
			}
			page, err := cli.ListPlanVCSBranches(cmd.Context(), args[0], opts)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(vcsBranchList(page.Results))
		},
	}
	c.Flags().IntVar(&limit, "limit", 0, "max items when --all is set (0 = unlimited)")
	c.Flags().IntVar(&max, "max-results", 25, "page size")
	c.Flags().BoolVar(&all, "all", false, "paginate through all results")
	return c
}
