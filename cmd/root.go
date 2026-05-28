package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	agentcmd "github.com/rahadiangg/bbx/cmd/agent"
	authcmd "github.com/rahadiangg/bbx/cmd/auth"
	buildcmd "github.com/rahadiangg/bbx/cmd/build"
	"github.com/rahadiangg/bbx/cmd/cmdctx"
	configcmd "github.com/rahadiangg/bbx/cmd/config"
	deploymentcmd "github.com/rahadiangg/bbx/cmd/deployment"
	futurecmd "github.com/rahadiangg/bbx/cmd/future"
	plancmd "github.com/rahadiangg/bbx/cmd/plan"
	projectcmd "github.com/rahadiangg/bbx/cmd/project"
	queuecmd "github.com/rahadiangg/bbx/cmd/queue"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

const longDesc = `bbx is a command-line interface for Atlassian Bamboo Server.

The MVP focuses on pipeline management: plans, builds, queue, and deployments.
Other Bamboo API areas (permissions, users, system admin, triggers, ...) are
recognized as commands but emit a "not yet implemented" notice.

Authentication uses a Bamboo Personal Access Token (PAT). Run 'bbx auth login'
to configure your server URL and token.`

// New returns the root Cobra command. It is exported for godoc + testability.
func New() *cobra.Command {
	g := cmdctx.Globals{}
	rootCmd := &cobra.Command{
		Use:           "bbx",
		Short:         "Atlassian Bamboo CLI",
		Long:          longDesc,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			f, err := output.ParseFormat(g.FormatFlag)
			if err != nil {
				return fail.New("invalid_flag", err.Error(), fail.ExitUsage)
			}
			g.Format = f
			cmdctx.Set(g)
			return nil
		},
	}

	pf := rootCmd.PersistentFlags()
	pf.StringVar(&g.ConfigPath, "config", "", "config file path (default: $XDG_CONFIG_HOME/bbx/config.yaml)")
	pf.StringVar(&g.ContextName, "context", "", "override the current context")
	pf.StringVarP(&g.FormatFlag, "output", "o", "", "output format: table|json|yaml (auto when unset)")
	pf.CountVarP(&g.Verbose, "verbose", "v", "increase verbosity (-v info, -vv debug)")

	// Subtree wiring
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(authcmd.New())
	rootCmd.AddCommand(agentcmd.New())
	rootCmd.AddCommand(configcmd.New())
	rootCmd.AddCommand(projectcmd.New())
	rootCmd.AddCommand(plancmd.New())
	rootCmd.AddCommand(buildcmd.New())
	rootCmd.AddCommand(queuecmd.New())
	rootCmd.AddCommand(deploymentcmd.New())
	for _, c := range futurecmd.All() {
		rootCmd.AddCommand(c)
	}

	return rootCmd
}

// Execute runs the root command and returns an appropriate exit code.
func Execute() int {
	rootCmd := New()
	if err := rootCmd.Execute(); err != nil {
		output.PrintError(cmdctx.G().Format, err)
		return exitCodeFor(err)
	}
	return fail.ExitOK
}

// exitCodeFor returns the exit code for an error returned from cobra.Execute.
// Errors of type *fail.Error already carry their intended exit code; anything
// else coming back from cobra is a usage problem (bad arg count, unknown
// flag, unknown command) and maps to ExitUsage.
func exitCodeFor(err error) int {
	var fe *fail.Error
	if errors.As(err, &fe) {
		return fail.ExitCode(err)
	}
	return fail.ExitUsage
}

// helper for code that needs to interact with stderr directly
func warn(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) }

// silence unused-import warning if warn is never used
var _ = warn
