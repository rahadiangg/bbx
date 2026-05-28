// Package agent implements the `bbx agent ...` command subtree — currently
// just `skills`, but the name reserves the namespace for future agent-side
// tooling (mcp adapters, agent-runtime helpers, etc.).
package agent

import (
	"github.com/spf13/cobra"

	skillscmd "github.com/rahadiangg/bbx/cmd/agent/skills"
)

// New returns the `bbx agent` parent command.
func New() *cobra.Command {
	c := &cobra.Command{
		Use:   "agent",
		Short: "AI-agent tooling — manage the embedded skill bundle, etc.",
	}
	c.AddCommand(skillscmd.New())
	return c
}
