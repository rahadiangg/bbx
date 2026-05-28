package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print bbx version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("bbx %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
			return nil
		},
	}
}
