package deployment

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/fail"
)

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "deployment",
		Aliases: []string{"deployments", "deploy"},
		Short:   "Trigger and inspect deployments",
	}
	c.AddCommand(newQueueCmd())
	c.AddCommand(newTriggerCmd())
	c.AddCommand(newCancelCmd())
	c.AddCommand(newResultCmd())
	c.AddCommand(newPreviewCmd())
	return c
}

func newQueueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "queue",
		Short: "List queued and in-progress deployments",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			env, err := cli.ListDeploymentQueue(cmd.Context())
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(env)
		},
	}
}

func newTriggerCmd() *cobra.Command {
	var (
		envID int64
		verID int64
	)
	c := &cobra.Command{
		Use:   "trigger",
		Short: "Trigger a deployment of a version to an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 || verID == 0 {
				return fail.New("missing_flag", "--environment-id and --version-id are required", fail.ExitUsage)
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			dr, err := cli.TriggerDeployment(cmd.Context(), envID, verID)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(dr)
		},
	}
	c.Flags().Int64Var(&envID, "environment-id", 0, "deployment environment ID")
	c.Flags().Int64Var(&verID, "version-id", 0, "deployment version ID")
	return c
}

func newCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <deployment-result-id>",
		Short: "Cancel a queued deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fail.New("invalid_arg", "id must be an integer", fail.ExitUsage)
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			if err := cli.CancelDeployment(cmd.Context(), id); err != nil {
				return err
			}
			cmdctx.G().Stderr("Cancelled deployment %d", id)
			return nil
		},
	}
}

func newResultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "result <deployment-result-id>",
		Short: "Get a deployment result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fail.New("invalid_arg", "id must be an integer", fail.ExitUsage)
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			dr, err := cli.GetDeploymentResult(cmd.Context(), id)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(dr)
		},
	}
}

func newPreviewCmd() *cobra.Command {
	var (
		projectID     string
		planResultKey string
	)
	c := &cobra.Command{
		Use:   "preview",
		Short: "Preview the next deployment version",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			vp, err := cli.PreviewDeploymentVersion(cmd.Context(), projectID, planResultKey)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(vp)
		},
	}
	c.Flags().StringVar(&projectID, "project-id", "", "deployment project ID")
	c.Flags().StringVar(&planResultKey, "plan-result-key", "", "plan result key (e.g. PROJ-PLAN-42)")
	return c
}
