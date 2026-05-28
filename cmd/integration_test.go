package cmd

import (
	"strings"
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

func TestVersionCommand(t *testing.T) {
	res := runCmdNoConfig(t, "version")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit = %d\nstderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "bbx") {
		t.Fatalf("stdout missing bbx: %q", res.Stdout)
	}
}

func TestRootHelpListsAllSubtrees(t *testing.T) {
	res := runCmdNoConfig(t, "--help")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit = %d", res.ExitCode)
	}
	for _, want := range []string{"auth", "config", "plan", "build", "queue", "deployment", "permissions"} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("help missing %q\n%s", want, res.Stdout)
		}
	}
}

func TestNoConfigEmitsStructuredError(t *testing.T) {
	res := runCmdNoConfig(t, "plan", "list")
	// "no contexts configured" is surfaced from cmdctx as a plain error; the
	// root command maps it to ExitUsage via exitCodeFor (anything not a
	// *fail.Error is a usage/setup problem).
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit = %d, want %d", res.ExitCode, fail.ExitUsage)
	}
	if !strings.Contains(res.Stderr, "no contexts configured") {
		t.Fatalf("stderr = %q", res.Stderr)
	}
}

func TestFutureStubExitsWithNotImpl(t *testing.T) {
	res := runCmdNoConfig(t, "permissions")
	if res.ExitCode != fail.ExitNotImpl {
		t.Fatalf("exit = %d, want %d\nstderr=%s", res.ExitCode, fail.ExitNotImpl, res.Stderr)
	}
	if !strings.Contains(res.Stderr, "not_implemented") {
		t.Fatalf("stderr missing not_implemented: %q", res.Stderr)
	}
}

func TestInvalidOutputFormat(t *testing.T) {
	res := runCmdNoConfig(t, "-o", "xml", "version")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit = %d, want %d", res.ExitCode, fail.ExitUsage)
	}
}

func TestAuthWhoami(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/currentUser", 200, `{"name":"jdoe","fullName":"Jane Doe"}`)
	res := runCmdEnv(t, fb, "auth", "whoami")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit = %d\nstderr=%s", res.ExitCode, res.Stderr)
	}
	var u map[string]string
	mustDecodeJSON(t, res.Stdout, &u)
	if u["name"] != "jdoe" || u["fullName"] != "Jane Doe" {
		t.Fatalf("user = %+v", u)
	}
}

func TestAuthLoginWritesConfig(t *testing.T) {
	fb := newFakeBamboo(t)
	dir := t.TempDir()
	cfg := dir + "/config.yaml"
	t.Setenv("BBX_CONFIG", cfg)
	t.Setenv("BBX_AGENT_MODE", "1")
	t.Setenv("BBX_TOKEN", "")
	res := runCmd(t,
		"auth", "login",
		"--name", "alt",
		"--base-url", fb.URL(),
		"--token", "the-pat",
	)
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit = %d\nstderr=%s", res.ExitCode, res.Stderr)
	}
	b, err := readFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b, "the-pat") || !strings.Contains(b, "alt") {
		t.Fatalf("config = %s", b)
	}
}

func TestAuthLoginMissingFlagsNonInteractive(t *testing.T) {
	// stdin is not a TTY in the test harness, so prompting is impossible.
	dir := t.TempDir()
	t.Setenv("BBX_CONFIG", dir+"/c.yaml")
	t.Setenv("BBX_AGENT_MODE", "1")
	t.Setenv("BBX_TOKEN", "")
	res := runCmd(t, "auth", "login")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit = %d, want usage\nstderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestAuthLogout(t *testing.T) {
	fb := newFakeBamboo(t)
	_ = fb // unused
	dir := t.TempDir()
	cfg := dir + "/config.yaml"
	writeConfigFile(t, cfg, "https://x", "t")
	t.Setenv("BBX_CONFIG", cfg)
	t.Setenv("BBX_AGENT_MODE", "1")
	t.Setenv("BBX_TOKEN", "")
	res := runCmd(t, "auth", "logout")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit = %d\nstderr=%s", res.ExitCode, res.Stderr)
	}
	b, _ := readFile(cfg)
	if strings.Contains(b, "default:") {
		t.Fatalf("default context should be gone: %s", b)
	}
}

// readFile is a tiny wrapper for table-test ergonomics.
func readFile(path string) (string, error) {
	b, err := readFileBytes(path)
	return string(b), err
}
