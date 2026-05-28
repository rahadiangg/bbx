// Package project implements the `bbx project ...` command subtree —
// project-scoped reads (metadata, variables, repositories, bulk Specs export).
// Project is the parent of plans in Bamboo's domain model.
package project

import (
	"github.com/spf13/cobra"

	repocmd "github.com/rahadiangg/bbx/cmd/project/repository"
	varcmd "github.com/rahadiangg/bbx/cmd/project/variable"
)

// New returns the `bbx project` parent command.
func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "project",
		Aliases: []string{"projects"},
		Short:   "Inspect Bamboo projects (parents of plans)",
	}
	c.AddCommand(newGetCmd())
	c.AddCommand(newSpecCmd())
	c.AddCommand(varcmd.New())
	c.AddCommand(repocmd.New())
	return c
}
