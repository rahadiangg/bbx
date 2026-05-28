package variable

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

type varList []api.PlanVariable

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
		Short:   "Manage plan variables",
	}
	c.AddCommand(newListCmd())
	c.AddCommand(newGetCmd())
	c.AddCommand(newSetCmd())
	c.AddCommand(newDeleteCmd())
	return c
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <plan-key>",
		Short: "List plan variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			vs, err := cli.ListPlanVariables(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(varList(vs))
		},
	}
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <plan-key> <name>",
		Short: "Get a single plan variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			v, err := cli.GetPlanVariable(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(v)
		},
	}
}

func newSetCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "set <plan-key> <name> <value>",
		Short: "Create or update a plan variable",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			// Try update; if not found, fall back to create.
			v, err := cli.UpdatePlanVariable(ctx, args[0], args[1], args[2])
			if err != nil {
				if fe, ok := err.(*fail.Error); ok && fe.HTTPStatus == 404 {
					v, err = cli.AddPlanVariable(ctx, args[0], args[1], args[2])
				}
			}
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(v)
		},
	}
	return c
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <plan-key> <name>",
		Short: "Delete a plan variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.DeletePlanVariable(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Deleted variable %s on %s", args[1], args[0])
			return nil
		},
	}
}
