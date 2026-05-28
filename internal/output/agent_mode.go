package output

import (
	"os"

	"golang.org/x/term"
)

// IsAgentMode reports whether we should default to machine-readable JSON output.
// True when stdout is not a TTY or when known AI-agent environment variables are set.
func IsAgentMode() bool {
	if os.Getenv("BBX_AGENT_MODE") != "" {
		return true
	}
	if os.Getenv("CLAUDECODE") != "" || os.Getenv("CLAUDE_CODE") != "" {
		return true
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return true
	}
	return !term.IsTerminal(int(os.Stdout.Fd()))
}
