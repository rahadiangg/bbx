package plan

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

// newConfigCmd implements `bbx plan config <key>` — the structured JSON
// counterpart to `bbx plan spec`. Useful when an AI agent wants to inspect
// stages/jobs programmatically without parsing Java.
func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config <plan-key>",
		Short: "Get the expanded plan configuration as structured JSON",
		Long: `Fetch the plan with maximum expand (stages, jobs, actions,
variables, branches) as structured JSON. Complementary to 'bbx plan spec':
the spec endpoint returns Bamboo Specs Java source, this one returns the
same information as nested JSON for easier programmatic inspection.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			cfg, err := cli.GetPlanConfig(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(cfg)
		},
	}
}
