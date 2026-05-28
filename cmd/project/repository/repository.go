// Package repository implements `bbx project repository list` — the repos
// linked at the project level (inherited by every plan in the project).
package repository

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

type repoList []api.ProjectRepository

func (l repoList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "ID", "NAME", "DESCRIPTION")
	for _, r := range l {
		t.AppendRow([]any{r.ID, r.Name, r.Description})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "repository",
		Aliases: []string{"repositories", "repo", "repos"},
		Short:   "Inspect project-linked repositories",
	}
	c.AddCommand(newListCmd())
	return c
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <project-key>",
		Short: "List repositories linked to a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			rs, err := cli.ListProjectRepositories(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(repoList(rs))
		},
	}
}
