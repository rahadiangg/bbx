// Package variable implements `bbx project variable list/get` — list-only;
// project variable writes are out of scope for the current iteration
// (the POST endpoint takes query params, mirroring the plan variable quirk).
package variable

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

type varList []api.ProjectVariable

func (l varList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "NAME", "VALUE")
	for _, v := range l {
		t.AppendRow([]any{v.Name, v.Value})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "variable",
		Aliases: []string{"variables", "var", "vars"},
		Short:   "Inspect project-scoped variables",
		Long: `Project variables are inherited by every plan in the project.
Sensitive values are masked server-side as '********' — bbx surfaces
them as-is; you'll need to re-enter actual secrets at the target system
if migrating.`,
	}
	c.AddCommand(newListCmd())
	c.AddCommand(newGetCmd())
	return c
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <project-key>",
		Short: "List project-scoped variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			vs, err := cli.ListProjectVariables(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(varList(vs))
		},
	}
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <project-key> <name>",
		Short: "Get a single project variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			v, err := cli.GetProjectVariable(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(v)
		},
	}
}
