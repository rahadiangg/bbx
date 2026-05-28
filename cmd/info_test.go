package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

func TestInfoCommandHappyPath(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/info", 200,
		`{"version":"8.2.4","edition":"","buildNumber":"80210","state":"RUNNING"}`)
	res := runCmdEnv(t, fb, "info")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	for _, want := range []string{"8.2.4", "80210", "RUNNING"} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("output missing %q\n%s", want, res.Stdout)
		}
	}
}

func TestInfoUpdatesCachedVersion(t *testing.T) {
	fb := newFakeBamboo(t)
	// First call returns 8.2.4 — captured at login.
	fb.expect("GET", "/rest/api/latest/info", 200, `{"version":"8.2.4","buildNumber":"80210"}`)
	loginRes := runCmd(t, "auth", "login",
		"--name", "default", "--base-url", fb.URL(), "--token", "t")
	_ = loginRes // BBX_CONFIG/BBX_AGENT_MODE are set in runCmdEnv only; below we follow up via runCmd

	// The integration test for login already covers config writing; here we
	// re-use the simpler runCmdEnv harness which writes its own minimal
	// config pointing at fb. We then verify `bbx info` overwrites the
	// version field if the server reports a newer one.
	fb2 := newFakeBamboo(t)
	fb2.expect("GET", "/rest/api/latest/info", 200, `{"version":"9.6.1","buildNumber":"90600"}`)
	res := runCmdEnv(t, fb2, "info")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "9.6.1") {
		t.Fatalf("expected refreshed version in output: %s", res.Stdout)
	}

	// Read back the persisted config from BBX_CONFIG; runCmdEnv sets that env
	// var to a tempfile, so we read whatever is current.
	cfgPath := os.Getenv("BBX_CONFIG")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(b), "server-version: 9.6.1") {
		t.Fatalf("cached version not updated; config:\n%s", b)
	}
}

func TestInfoNoUpdateFlagSkipsConfigWrite(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/info", 200, `{"version":"9.0.0","buildNumber":"90000"}`)
	res := runCmdEnv(t, fb, "info", "--no-update")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	cfgPath := os.Getenv("BBX_CONFIG")
	b, _ := os.ReadFile(cfgPath)
	if strings.Contains(string(b), "server-version: 9.0.0") {
		t.Fatalf("--no-update should not have persisted; config:\n%s", b)
	}
}

func TestAuthLoginCapturesServerVersion(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/info", 200,
		`{"version":"8.2.4","edition":"","buildNumber":"80210","state":"RUNNING"}`)
	dir := t.TempDir()
	cfg := dir + "/c.yaml"
	t.Setenv("BBX_CONFIG", cfg)
	t.Setenv("BBX_AGENT_MODE", "1")
	t.Setenv("BBX_TOKEN", "")
	res := runCmd(t, "auth", "login",
		"--name", "default", "--base-url", fb.URL(), "--token", "t")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	b, _ := os.ReadFile(cfg)
	if !strings.Contains(string(b), "server-version: 8.2.4") {
		t.Fatalf("config missing captured version:\n%s", b)
	}
	if !strings.Contains(string(b), "server-build: \"80210\"") {
		// YAML may quote or not-quote depending on the marshaler; accept either form
		if !strings.Contains(string(b), "server-build: 80210") {
			t.Fatalf("config missing captured build number:\n%s", b)
		}
	}
	if !strings.Contains(res.Stderr, "Bamboo 8.2.4") {
		t.Fatalf("stderr should mention captured version: %q", res.Stderr)
	}
}

func TestAuthLoginSucceedsWhenInfoFails(t *testing.T) {
	fb := newFakeBamboo(t)
	// /info returns 500 — login should still save the context.
	fb.expect("GET", "/rest/api/latest/info", 500, `{"message":"boom"}`)
	dir := t.TempDir()
	cfg := dir + "/c.yaml"
	t.Setenv("BBX_CONFIG", cfg)
	t.Setenv("BBX_AGENT_MODE", "1")
	t.Setenv("BBX_TOKEN", "")
	res := runCmd(t, "auth", "login",
		"--name", "default", "--base-url", fb.URL(), "--token", "t")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stderr, "warning") {
		t.Errorf("expected warning about info failure: %q", res.Stderr)
	}
	b, _ := os.ReadFile(cfg)
	if !strings.Contains(string(b), "token: t") {
		t.Fatalf("token not persisted despite info failure:\n%s", b)
	}
}
