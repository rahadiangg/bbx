package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/api"
	"github.com/rahadiangg/bbx/internal/config"
	"github.com/rahadiangg/bbx/internal/fail"
)

func newLoginCmd() *cobra.Command {
	var (
		contextName string
		baseURL     string
		token       string
		insecure    bool
	)
	c := &cobra.Command{
		Use:   "login",
		Short: "Configure a Bamboo context (URL + Personal Access Token)",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)
			interactive := term.IsTerminal(int(os.Stdin.Fd()))

			if contextName == "" {
				contextName = "default"
				if interactive {
					contextName = promptDefault(reader, "Context name", "default")
				}
			}
			if baseURL == "" {
				if !interactive {
					return fail.New("missing_flag", "--base-url is required in non-interactive mode", fail.ExitUsage)
				}
				baseURL = promptRequired(reader, "Bamboo base URL (e.g. https://bamboo.example.com)")
			}
			baseURL = strings.TrimRight(baseURL, "/")

			if token == "" {
				if !interactive {
					return fail.New("missing_flag", "--token is required in non-interactive mode", fail.ExitUsage)
				}
				fmt.Print("Personal Access Token: ")
				b, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println()
				if err != nil {
					return fail.Wrap(err, "read_token", fail.ExitGeneric)
				}
				token = strings.TrimSpace(string(b))
				if token == "" {
					return fail.New("missing_token", "token is required", fail.ExitUsage)
				}
			}

			cfg, err := cmdctx.G().Config()
			if err != nil {
				return err
			}
			if cfg.Contexts == nil {
				cfg.Contexts = map[string]config.Context{}
			}
			ctx := config.Context{
				BaseURL:            baseURL,
				Token:              token,
				InsecureSkipVerify: insecure,
			}

			// Probe /info before persisting. Best-effort: if the server is
			// unreachable or returns an error we still save the credentials
			// (the user can run `bbx info` later to refresh). This avoids
			// blocking login on a transient network blip.
			if cli, err := api.New(api.Options{BaseURL: baseURL, Token: token, InsecureSkipVerify: insecure}); err == nil {
				if info, err := cli.GetServerInfo(cmd.Context()); err == nil {
					ctx.ServerVersion = info.Version
					ctx.ServerBuildNumber = info.BuildNumber
					ctx.ServerEdition = info.Edition
				} else {
					cmdctx.G().Stderr("warning: could not fetch server info (%v); login saved anyway", err)
				}
			}

			cfg.Contexts[contextName] = ctx
			if cfg.CurrentContext == "" {
				cfg.CurrentContext = contextName
			}
			if err := cmdctx.G().SaveConfig(); err != nil {
				return fail.Wrap(err, "save_config", fail.ExitGeneric)
			}
			if ctx.ServerVersion != "" {
				cmdctx.G().Stderr("Saved context %q (current: %s, server: Bamboo %s)", contextName, cfg.CurrentContext, ctx.ServerVersion)
			} else {
				cmdctx.G().Stderr("Saved context %q (current: %s)", contextName, cfg.CurrentContext)
			}
			return nil
		},
	}
	c.Flags().StringVar(&contextName, "name", "", "context name (default: \"default\")")
	c.Flags().StringVar(&baseURL, "base-url", "", "Bamboo server base URL")
	c.Flags().StringVar(&token, "token", "", "Personal Access Token (omit to prompt)")
	c.Flags().BoolVar(&insecure, "insecure", false, "skip TLS certificate verification")
	return c
}

func promptRequired(r *bufio.Reader, label string) string {
	for {
		fmt.Print(label + ": ")
		line, _ := r.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
}

func promptDefault(r *bufio.Reader, label, def string) string {
	fmt.Printf("%s [%s]: ", label, def)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}
