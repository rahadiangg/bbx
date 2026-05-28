package comment

import (
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

type commentList []api.BuildComment

func (c commentList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "ID", "AUTHOR", "CREATED", "CONTENT")
	for _, bc := range c {
		t.AppendRow([]any{bc.ID, bc.Author, bc.CreationDate, bc.Content})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "comment",
		Aliases: []string{"comments"},
		Short:   "Manage build comments",
	}
	c.AddCommand(newListCmd(), newAddCmd(), newDeleteCmd())
	return c
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <build-key>",
		Short: "List comments on a build",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			items, err := cli.ListBuildComments(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(commentList(items))
		},
	}
}

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <build-key> <content>",
		Short: "Add a comment to a build",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			// Bamboo returns 204 No Content; surface a stderr confirmation
			// instead of emitting a zero-valued struct.
			if err := cli.AddBuildComment(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Added comment on %s", args[0])
			return nil
		},
	}
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <build-key> <comment-id>",
		Short: "Delete a comment from a build",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			id, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fail.New("invalid_arg", "comment id must be an integer", fail.ExitUsage)
			}
			if err := cli.DeleteBuildComment(cmd.Context(), args[0], id); err != nil {
				return err
			}
			cmdctx.G().Stderr("Deleted comment %d on %s", id, args[0])
			return nil
		},
	}
}
