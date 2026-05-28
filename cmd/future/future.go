// Package future registers placeholder commands for Bamboo API areas that are
// outside the MVP scope. Each command emits a clear "not yet implemented" notice
// pointing at docs/API_COVERAGE.md and exits with the dedicated ExitNotImpl code.
package future

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/internal/fail"
)

type stub struct {
	use   string
	short string
}

var stubs = []stub{
	{"permissions", "(future) manage Bamboo permissions"},
	{"users", "(future) manage Bamboo users and groups"},
	{"system", "(future) read system information"},
	{"triggers", "(future) manage plan triggers"},
	{"trusted-keys", "(future) manage trusted SSH/SSL keys"},
	{"session", "(future) inspect session information"},
	{"avatars", "(future) manage user/project avatars"},
}

// All returns one Cobra command per stubbed Bamboo area.
func All() []*cobra.Command {
	out := make([]*cobra.Command, 0, len(stubs))
	for _, s := range stubs {
		s := s
		c := &cobra.Command{
			Use:   s.use,
			Short: s.short,
			Long:  "This command group is not yet implemented in bbx.\nSee docs/API_COVERAGE.md for the scope plan.",
			RunE: func(cmd *cobra.Command, args []string) error {
				return fail.New("not_implemented",
					"command group '"+s.use+"' is not yet implemented; see docs/API_COVERAGE.md",
					fail.ExitNotImpl)
			},
		}
		out = append(out, c)
	}
	return out
}
