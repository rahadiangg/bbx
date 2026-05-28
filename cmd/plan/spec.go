package plan

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

// rawSpec wraps the PlanSpec for table-mode rendering: writes the Java source
// as plain text so users can `bbx plan spec X > X.java`. JSON/YAML modes fall
// through to the default marshaller and emit the full envelope.
type rawSpec api.PlanSpec

func (r rawSpec) RenderTable(w io.Writer) error {
	_, err := io.WriteString(w, r.Code)
	if err != nil {
		return err
	}
	if len(r.Code) > 0 && r.Code[len(r.Code)-1] != '\n' {
		_, _ = io.WriteString(w, "\n")
	}
	return nil
}

func newSpecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "spec <plan-key>",
		Short: "Print the Bamboo Specs Java source for a plan",
		Long: `Fetch the Bamboo Specs Java source for a plan (the complete
configuration as executable Java — stages, jobs, tasks, variables,
branch rules, permissions).

Bamboo auto-generates this for every plan, including UI-built ones —
it does NOT require the plan to have been created from Specs.

Default output mode writes the Java source as plain text, so:

  bbx plan spec PROJ-PLAN > PROJ-PLAN.java

With -o json or -o yaml, the full envelope (projectKey, buildKey, code)
is emitted as structured output for downstream tooling / AI agents.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			spec, err := cli.GetPlanSpec(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			// Structured modes: emit the envelope as-is.
			// Table mode: write raw Java via the Renderable above.
			f := cmdctx.G().Format
			if f == output.FormatTable {
				return output.Print(f, rawSpec(*spec))
			}
			return cmdctx.G().Emit(spec)
		},
		Example: `  bbx plan spec PROJ-PLAN
  bbx plan spec PROJ-PLAN -o json
  bbx plan spec PROJ-PLAN > PROJ-PLAN.java`,
	}
}
