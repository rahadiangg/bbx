package build

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <build-result-key>",
		Short: "Stop a queued or running build (e.g. PROJ-PLAN-42)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.StopBuild(cmd.Context(), args[0]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Stopped build %s", args[0])
			return nil
		},
	}
}

func newContinueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "continue <build-result-key>",
		Short: "Continue a stopped build",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.ContinueBuild(cmd.Context(), args[0]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Resumed build %s", args[0])
			return nil
		},
	}
}
