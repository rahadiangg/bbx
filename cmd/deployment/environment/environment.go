// Package environment implements `bbx deployment environment ...` — config
// extraction for a deployment environment (variables, requirements,
// agent assignments).
package environment

import (
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

func New() *cobra.Command {
	c := &cobra.Command{
		Use:     "environment",
		Aliases: []string{"environments", "env"},
		Short:   "Inspect deployment environments",
	}
	c.AddCommand(newGetCmd())
	c.AddCommand(newVariableCmd())
	c.AddCommand(newRequirementCmd())
	c.AddCommand(newAgentCmd())
	return c
}

func parseEnvID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fail.New("invalid_arg", "environment id must be an integer", fail.ExitUsage)
	}
	return id, nil
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <environment-id>",
		Short: "Get deployment environment metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseEnvID(args[0])
			if err != nil {
				return err
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			e, err := cli.GetDeploymentEnvironment(cmd.Context(), id)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(e)
		},
	}
}

type envVarList []api.EnvVariable

func (l envVarList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "ID", "KEY", "VALUE")
	for _, v := range l {
		t.AppendRow([]any{v.ID, v.Key, v.Value})
	}
	t.Render()
	return nil
}

func newVariableCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "variable",
		Aliases: []string{"variables", "var", "vars"},
		Short:   "Inspect environment-scoped variables",
	}
	c.AddCommand(&cobra.Command{
		Use:   "list <environment-id>",
		Short: "List variables defined on a deployment environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseEnvID(args[0])
			if err != nil {
				return err
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			vs, err := cli.ListEnvironmentVariables(cmd.Context(), id)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(envVarList(vs))
		},
	})
	return c
}

type reqList []api.EnvRequirement

func (l reqList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "ID", "KEY", "MATCH TYPE", "MATCH VALUE")
	for _, r := range l {
		t.AppendRow([]any{r.ID, r.Key, r.MatchType, r.MatchValue})
	}
	t.Render()
	return nil
}

func newRequirementCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "requirement",
		Aliases: []string{"requirements", "req", "reqs"},
		Short:   "Inspect agent-capability requirements for an environment",
	}
	c.AddCommand(&cobra.Command{
		Use:   "list <environment-id>",
		Short: "List agent-capability requirements",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseEnvID(args[0])
			if err != nil {
				return err
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			rs, err := cli.ListEnvironmentRequirements(cmd.Context(), id)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(reqList(rs))
		},
	})
	return c
}

type agentList []api.AgentAssignment

func (l agentList) RenderTable(w io.Writer) error {
	t := output.NewTable(w, "EXECUTOR ID", "EXECUTOR TYPE", "EXECUTABLE ID", "EXECUTABLE TYPE")
	for _, a := range l {
		t.AppendRow([]any{a.ExecutorID, a.ExecutorType, a.ExecutableID, a.ExecutableType})
	}
	t.Render()
	return nil
}

func newAgentCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "agent",
		Aliases: []string{"agents"},
		Short:   "Inspect agents/Docker images assigned to an environment",
	}
	c.AddCommand(&cobra.Command{
		Use:   "list <environment-id>",
		Short: "List agent assignments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseEnvID(args[0])
			if err != nil {
				return err
			}
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			as, err := cli.ListEnvironmentAgentAssignments(cmd.Context(), id)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(agentList(as))
		},
	})
	return c
}
