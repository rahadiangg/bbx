// Package project implements `bbx deployment project ...` — config-extraction
// for deployment projects (the deploy-side parent of environments+versions).
package project

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	repocmd "github.com/rahadiangg/bbx/cmd/deployment/project/repository"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

type dpList []api.DeploymentProject

func (l dpList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "ID", "NAME", "PLAN KEY", "ENVS")
	for _, dp := range l {
		planKey := ""
		if dp.PlanKey != nil {
			if v, ok := dp.PlanKey["key"].(string); ok {
				planKey = v
			}
		}
		t.AppendRow([]any{dp.ID, dp.Name, planKey, len(dp.Environments)})
	}
	t.Render()
	return nil
}

func New() *cobra.Command {
	c := &cobra.Command{
		Use:   "project",
		Short: "Inspect deployment projects (deploy-side parents)",
	}
	c.AddCommand(newListCmd())
	c.AddCommand(newGetCmd())
	c.AddCommand(newSpecCmd())
	c.AddCommand(repocmd.New())
	return c
}

func newListCmd() *cobra.Command {
	var forPlan string
	c := &cobra.Command{
		Use:   "list",
		Short: "List deployment projects (optionally filtered to a single plan)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			var dps []api.DeploymentProject
			if forPlan != "" {
				dps, err = cli.ListDeploymentProjectsForPlan(cmd.Context(), forPlan)
			} else {
				dps, err = cli.ListDeploymentProjects(cmd.Context())
			}
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(dpList(dps))
		},
	}
	c.Flags().StringVar(&forPlan, "for-plan", "", "only list deployment projects linked to this plan key")
	return c
}

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fail.New("invalid_arg", "id must be an integer: "+s, fail.ExitUsage)
	}
	return id, nil
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <deployment-project-id>",
		Short: "Get full deployment project config (incl. environments)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			dp, err := cli.GetDeploymentProject(cmd.Context(), id)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(dp)
		},
	}
}

type rawDeploymentSpec api.DeploymentProjectSpec

func (r rawDeploymentSpec) RenderTable(w io.Writer) error {
	fmt.Fprintf(w, "// === deployment project %d ===\n", r.DeploymentID)
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
		Use:   "spec <deployment-project-id>",
		Short: "Print the Bamboo Specs Java source for a deployment project",
		Long: `Bamboo Specs Java source for the deployment project (environments,
release versioning, agent assignments). Same idea as 'bbx plan spec' but
for the deploy side. Default output is raw Java; -o json/yaml emits the
{deploymentId, code} envelope.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			sp, err := cli.GetDeploymentProjectSpec(cmd.Context(), id)
			if err != nil {
				return err
			}
			if cmdctx.G().Format == output.FormatTable {
				return output.Print(cmdctx.G().Format, rawDeploymentSpec(*sp))
			}
			return cmdctx.G().Emit(sp)
		},
	}
}
