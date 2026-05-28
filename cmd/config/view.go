package config

import (
	"github.com/spf13/cobra"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
)

func newViewCmd() *cobra.Command {
	var showSecrets bool
	c := &cobra.Command{
		Use:   "view",
		Short: "Print the current config (tokens redacted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cmdctx.G().Config()
			if err != nil {
				return err
			}
			redacted := *cfg
			contexts := make(map[string]any, len(cfg.Contexts))
			for k, v := range cfg.Contexts {
				m := map[string]any{
					"base-url": v.BaseURL,
				}
				if v.InsecureSkipVerify {
					m["insecure-skip-verify"] = true
				}
				if showSecrets {
					m["token"] = v.Token
				} else if v.Token != "" {
					m["token"] = "***"
				}
				if v.ServerVersion != "" {
					m["server-version"] = v.ServerVersion
				}
				if v.ServerBuildNumber != "" {
					m["server-build"] = v.ServerBuildNumber
				}
				if v.ServerEdition != "" {
					m["server-edition"] = v.ServerEdition
				}
				contexts[k] = m
			}
			return cmdctx.G().Emit(map[string]any{
				"current-context": redacted.CurrentContext,
				"contexts":        contexts,
			})
		},
	}
	c.Flags().BoolVar(&showSecrets, "show-secrets", false, "display tokens in cleartext")
	return c
}
