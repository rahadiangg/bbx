// Package repository implements `bbx deployment project repository list <id>`.
package repository

import (
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

type repoList []api.DeploymentProjectRepository

func (l repoList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "ID", "NAME")
	for _, r := range l {
		t.AppendRow([]any{r.ID, r.Name})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "repository",
		Aliases: []string{"repositories", "repo", "repos"},
		Short:   "Inspect deployment-project repositories",
	}
	c.AddCommand(newListCmd())
	return c
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <deployment-project-id>",
		Short: "List repositories linked to a deployment project",
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
			rs, err := cli.ListDeploymentProjectRepositories(cmd.Context(), id)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(repoList(rs))
		},
	}
}
