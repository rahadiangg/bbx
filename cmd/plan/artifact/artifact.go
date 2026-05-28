package artifact

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

type artifactList []api.PlanArtifact

func (a artifactList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "NAME", "LOCATION", "PATTERN", "SHARED", "REQUIRED")
	for _, ar := range a {
		t.AppendRow([]any{ar.Name, ar.Location, ar.CopyPattern, ar.Shared, ar.Required})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "artifact",
		Aliases: []string{"artifacts"},
		Short:   "Inspect plan artifact definitions",
	}
	c.AddCommand(newListCmd())
	return c
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <plan-key>",
		Short: "List artifact definitions for a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			page, err := cli.ListPlanArtifacts(cmd.Context(), args[0], api.PageOpts{})
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(artifactList(page.Results))
		},
	}
}
