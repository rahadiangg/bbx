package cmd

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

// newInfoCmd implements `bbx info`. Hits the live /rest/api/latest/info on
// the active context, prints the result, and refreshes the cached version in
// the config if it drifted from what was captured at login time.
func newInfoCmd() *cobra.Command {
	var noUpdate bool
	c := &cobra.Command{
		Use:   "info",
		Short: "Show the Bamboo server's version and build info",
		Long: `Fetches /rest/api/latest/info on the active Bamboo context and prints it.

bbx caches the server version in the config to gate version-specific
behavior without an extra round-trip per command. Running 'bbx info'
refreshes that cache.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cmdctx.G().Client(cmd.Context())
			if err != nil {
				return err
			}
			info, err := cli.GetServerInfo(cmd.Context())
			if err != nil {
				return err
			}

			if !noUpdate {
				// Refresh the cached fields on the active context if they drifted.
				cfg, cerr := cmdctx.G().Config()
				if cerr == nil && cfg != nil {
					name := cfg.CurrentContext
					if cmdctx.G().ContextName != "" {
						name = cmdctx.G().ContextName
					}
					if ctx, ok := cfg.Contexts[name]; ok {
						changed := ctx.ServerVersion != info.Version ||
							ctx.ServerBuildNumber != info.BuildNumber ||
							ctx.ServerEdition != info.Edition
						if changed {
							ctx.ServerVersion = info.Version
							ctx.ServerBuildNumber = info.BuildNumber
							ctx.ServerEdition = info.Edition
							cfg.Contexts[name] = ctx
							if serr := cmdctx.G().SaveConfig(); serr != nil {
								cmdctx.G().Stderr("warning: could not persist refreshed info: %v", serr)
							}
						}
					}
				}
			}
			return cmdctx.G().Emit(info)
		},
	}
	c.Flags().BoolVar(&noUpdate, "no-update", false, "do not refresh the cached version in the config")
	return c
}
