// Package version implements `bbx deployment version list <deployment-project-id>`.
package version

import (
	"context"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

type versionList []api.DeploymentVersion

func (l versionList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "ID", "NAME", "CREATED BY", "CREATED")
	for _, v := range l {
		t.AppendRow([]any{v.ID, v.Name, v.CreatorUserName, v.CreationDate})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "version",
		Aliases: []string{"versions"},
		Short:   "Inspect deployment versions (artifact snapshots)",
	}
	c.AddCommand(newListCmd())
	return c
}

func newListCmd() *cobra.Command {
	var (
		all   bool
		limit int
		max   int
	)
	c := &cobra.Command{
		Use:   "list <deployment-project-id>",
		Short: "List deployment versions for a deployment project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fail.New("invalid_arg", "id must be an integer", fail.ExitUsage)
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			opts := api.PageOpts{MaxResults: max}
			if all {
				items, err := api.Iterate(cmd.Context(), opts, limit,
					func(c2 context.Context, o api.PageOpts) (api.Page[api.DeploymentVersion], error) {
						return cli.ListDeploymentVersions(c2, id, o)
					})
				if err != nil {
					return err
				}
				return cmdctx.G().Emit(versionList(items))
			}
			page, err := cli.ListDeploymentVersions(cmd.Context(), id, opts)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(versionList(page.Results))
		},
	}
	c.Flags().IntVar(&limit, "limit", 0, "max items when --all is set")
	c.Flags().IntVar(&max, "max-results", 25, "page size")
	c.Flags().BoolVar(&all, "all", false, "paginate through all results")
	return c
}
