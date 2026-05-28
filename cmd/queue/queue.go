package queue

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

type queueList []api.QueuedBuild

func (q queueList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "PLAN", "BUILD#", "RESULT KEY", "REASON")
	for _, qb := range q {
		t.AppendRow([]any{qb.PlanKey, qb.BuildNumber, qb.BuildResultKey, qb.TriggerReason})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:   "queue",
		Short: "Inspect the Bamboo build queue",
	}
	c.AddCommand(newListCmd())
	return c
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List queued builds",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			page, err := cli.ListQueue(cmd.Context(), api.PageOpts{})
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(queueList(page.Results))
		},
	}
}
