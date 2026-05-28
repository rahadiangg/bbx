package label

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

type labelList []api.BuildLabel

func (l labelList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "NAME")
	for _, bl := range l {
		t.AppendRow([]any{bl.Name})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "label",
		Aliases: []string{"labels"},
		Short:   "Manage build labels",
	}
	c.AddCommand(newListCmd(), newAddCmd(), newDeleteCmd())
	return c
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <build-key>",
		Short: "List labels on a build",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			items, err := cli.ListBuildLabels(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(labelList(items))
		},
	}
}

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <build-key> <label>",
		Short: "Add a label to a build",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.AddBuildLabel(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Added label %q on %s", args[1], args[0])
			return nil
		},
	}
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <build-key> <label>",
		Short: "Remove a label from a build",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.DeleteBuildLabel(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Removed label %q from %s", args[1], args[0])
			return nil
		},
	}
}
