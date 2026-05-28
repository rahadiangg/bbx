package config

import "github.com/spf13/cobra"

func New() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "View and edit bbx configuration",
	}
	c.AddCommand(newViewCmd())
	c.AddCommand(newSetCmd())
	c.AddCommand(newUseContextCmd())
	c.AddCommand(newListContextsCmd())
	return c
}
