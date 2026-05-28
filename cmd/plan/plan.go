package plan

import (
	"github.com/spf13/cobra"

	branchcmd "github.com/rahadiangg/bbx/cmd/plan/branch"
	varcmd "github.com/rahadiangg/bbx/cmd/plan/variable"
)

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "plan",
		Aliases: []string{"plans"},
		Short:   "Manage Bamboo plans (pipelines)",
	}
	c.AddCommand(newListCmd())
	c.AddCommand(newGetCmd())
	c.AddCommand(newCloneCmd())
	c.AddCommand(newEnableCmd())
	c.AddCommand(newDisableCmd())
	c.AddCommand(newDeleteCmd())
	c.AddCommand(branchcmd.New())
	c.AddCommand(varcmd.New())
	return c
}
