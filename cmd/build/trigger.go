package build

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/fail"
)

func newTriggerCmd() *cobra.Command {
	var vars []string
	c := &cobra.Command{
		Use:   "trigger <plan-key>",
		Short: "Trigger a build of a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			vmap := map[string]string{}
			for _, v := range vars {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) != 2 {
					return fail.New("invalid_arg", "expected key=value: "+v, fail.ExitUsage)
				}
				vmap[parts[0]] = parts[1]
			}
			qb, err := cli.TriggerBuild(cmd.Context(), args[0], vmap)
			if err != nil {
				return err
			}
			return cmdctx.G().Emit(qb)
		},
	}
	c.Flags().StringArrayVar(&vars, "var", nil, "build variable key=value (repeatable)")
	return c
}
