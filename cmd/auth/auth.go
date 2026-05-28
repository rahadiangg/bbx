package auth

import "github.com/spf13/cobra"

// New returns the `bbx auth` command tree.
func New() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Manage Bamboo authentication",
	}
	c.AddCommand(newLoginCmd())
	c.AddCommand(newLogoutCmd())
	c.AddCommand(newWhoamiCmd())
	return c
}
