package build

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

// buildLogList is a Renderable that prints the logs of every job with a clear
// separator header in table mode. JSON/YAML modes use the BuildLog slice
// directly via reflection.
type buildLogList []api.BuildLog

func (l buildLogList) RenderTable(w io.Writer) error {
	for i, b := range l {
		if i > 0 {
			_, _ = io.WriteString(w, "\n")
		}
		fmt.Fprintf(w, "=== JOB %s ===\n", b.JobKey)
		_, _ = io.WriteString(w, b.Log)
		if !strings.HasSuffix(b.Log, "\n") {
			_, _ = io.WriteString(w, "\n")
		}
	}
	return nil
}

func newLogCmd() *cobra.Command {
	var jobFilter string
	c := &cobra.Command{
		Use:   "log <build-result-key>",
		Short: "Fetch the plain-text log of a build (best-effort)",
		Long: `Fetch the plain-text log for every job in a build.

CAVEAT: Bamboo Server's /download/* paths require session-cookie auth on
some versions (notably 8.2.4). Personal Access Tokens are silently
redirected to an HTML login page; bbx detects that case and exits with
'session_auth_required' (exit code 3). On newer Bamboo versions that
honour PAT on /download/, this command returns the logs as expected.

Use --job to filter to a single job key.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			logs, err := cli.GetBuildLog(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if jobFilter != "" {
				filtered := logs[:0]
				for _, l := range logs {
					if l.JobKey == jobFilter {
						filtered = append(filtered, l)
					}
				}
				logs = filtered
			}
			return output.Print(cmdctx.G().Format, buildLogList(logs))
		},
	}
	c.Flags().StringVar(&jobFilter, "job", "", "filter to a single job result key (e.g. PROJ-A-JOB1-42)")
	return c
}
