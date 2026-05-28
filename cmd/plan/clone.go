package plan

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

// newCloneCmd implements `bbx plan clone <src> <dst>`. This is the only path
// Bamboo Server exposes for creating a plan via REST — there is no
// `POST /plan` endpoint. The destination project must already exist.
func newCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <source-key> <destination-key>",
		Short: "Clone an existing plan into a new one",
		Long: `Create a new plan by cloning an existing one.

The destination project must already exist. The destination plan key must
not already exist — Bamboo returns 4xx otherwise.

Bamboo Server does not expose a "create plan from scratch" REST endpoint;
cloning is the only way to create a plan via this CLI.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := cli.ClonePlan(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			cmdctx.G().Stderr("Cloned %s -> %s", args[0], args[1])
			return cmdctx.G().Emit(result)
		},
	}
}
