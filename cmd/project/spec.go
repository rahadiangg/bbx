package project

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/output"
)

// rawProjectSpec renders the bulk Specs export as plain text, separating
// each plan's Java source with a clear delimiter.
type rawProjectSpec api.ProjectSpec

func (r rawProjectSpec) RenderTable(w io.Writer) error {
	for i, s := range r.Spec {
		if i > 0 {
			_, _ = io.WriteString(w, "\n")
		}
		fmt.Fprintf(w, "// === %s-%s ===\n", s.ProjectKey, s.BuildKey)
		_, _ = io.WriteString(w, s.Code)
		if len(s.Code) > 0 && s.Code[len(s.Code)-1] != '\n' {
			_, _ = io.WriteString(w, "\n")
		}
	}
	return nil
}

func newSpecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "spec <project-key>",
		Short: "Print the Bamboo Specs Java source for every plan in a project",
		Long: `Bulk Bamboo Specs export for a project — emits the Java source
for every plan in the project, concatenated with '// === KEY ===' headers
in table mode, or as a structured envelope with -o json/yaml.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			ps, err := cli.GetProjectSpec(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if cmdctx.G().Format == output.FormatTable {
				return output.Print(cmdctx.G().Format, rawProjectSpec(*ps))
			}
			return cmdctx.G().Emit(ps)
		},
	}
}
