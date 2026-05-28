package build

import (
	"io"

	"github.com/spf13/cobra"

	commentcmd "github.com/rahadiangg/bbx/cmd/build/comment"
	labelcmd "github.com/rahadiangg/bbx/cmd/build/label"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

type buildResultList []api.BuildResult

func (b buildResultList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "KEY", "STATE", "LIFECYCLE", "STARTED", "DURATION")
	for _, br := range b {
		t.AppendRow([]any{br.Key, br.State, br.LifeCycleState, br.BuildStartedTime, br.PrettyBuildDuration})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "build",
		Aliases: []string{"builds"},
		Short:   "Trigger, stop, and inspect builds",
	}
	c.AddCommand(newTriggerCmd())
	c.AddCommand(newStopCmd())
	c.AddCommand(newContinueCmd())
	c.AddCommand(newStatusCmd())
	c.AddCommand(newGetCmd())
	c.AddCommand(newHistoryCmd())
	c.AddCommand(newLatestCmd())
	c.AddCommand(newLogCmd())
	c.AddCommand(commentcmd.New())
	c.AddCommand(labelcmd.New())
	return c
}
