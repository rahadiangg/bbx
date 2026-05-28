package branch

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

type branchList []api.PlanBranch

func (b branchList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "KEY", "NAME", "ENABLED")
	for _, pb := range b {
		t.AppendRow([]any{pb.Key, pb.Name, pb.Enabled})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "branch",
		Aliases: []string{"branches"},
		Short:   "Manage plan branches",
	}
	c.AddCommand(newListCmd())
	c.AddCommand(newGetCmd())
	c.AddCommand(newCreateCmd())
	c.AddCommand(newDeleteCmd())
	c.AddCommand(newEnableCmd())
	c.AddCommand(newDisableCmd())
	return c
}

// newEnableCmd / newDisableCmd: plan branches *are* plans in Bamboo, so
// /plan/{branchKey}/enable accepts POST/DELETE just like for top-level plans.
// We expose ergonomic <plan-key> <branch-name> args and resolve the branch's
// own plan-key via GetPlanBranch — same two-step pattern as DeletePlanBranch.
func newEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <plan-key> <branch-name>",
		Short: "Enable a plan branch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			br, err := cli.GetPlanBranch(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if err := cli.EnablePlan(cmd.Context(), br.Key); err != nil {
				return err
			}
			cmdctx.G().Stderr("Enabled branch %s on %s (plan-key %s)", args[1], args[0], br.Key)
			return nil
		},
	}
}

func newDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <plan-key> <branch-name>",
		Short: "Disable a plan branch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			br, err := cli.GetPlanBranch(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if err := cli.DisablePlan(cmd.Context(), br.Key); err != nil {
				return err
			}
			cmdctx.G().Stderr("Disabled branch %s on %s (plan-key %s)", args[1], args[0], br.Key)
			return nil
		},
	}
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <plan-key> <branch-name>",
		Short: "Delete a plan branch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.DeletePlanBranch(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			cmdctx.G().Stderr("Deleted branch %s on %s", args[1], args[0])
			return nil
		},
	}
}

func newListCmd() *cobra.Command {
	var (
		all   bool
		limit int
		max   int
	)
	c := &cobra.Command{
		Use:   "list <plan-key>",
		Short: "List branches of a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			opts := api.PageOpts{MaxResults: max}
			if all {
				items, err := api.Iterate[api.PlanBranch](cmd.Context(), opts, limit,
					func(c2 context.Context, o api.PageOpts) (api.Page[api.PlanBranch], error) {
						return cli.ListPlanBranches(c2, args[0], o)
					})
				if err != nil {
					return err
				}
				return cmdctx.G().Emit(branchList(items))
			}
			page, err := cli.ListPlanBranches(cmd.Context(), args[0], opts)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(branchList(page.Results))
		},
	}
	c.Flags().IntVar(&limit, "limit", 0, "max items when --all is set")
	c.Flags().IntVar(&max, "max-results", 25, "page size")
	c.Flags().BoolVar(&all, "all", false, "paginate through all results")
	return c
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <plan-key> <branch-name>",
		Short: "Get a single plan branch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			b, err := cli.GetPlanBranch(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(b)
		},
	}
}

func newCreateCmd() *cobra.Command {
	var vcs string
	c := &cobra.Command{
		Use:   "create <plan-key> <branch-name>",
		Short: "Create a plan branch tracking a VCS branch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			b, err := cli.CreatePlanBranch(cmd.Context(), args[0], args[1], vcs)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(b)
		},
	}
	c.Flags().StringVar(&vcs, "vcs-branch", "", "VCS branch to track")
	return c
}
