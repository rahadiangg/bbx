package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

func TestConfigView(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "config", "view")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	var out map[string]any
	mustDecodeJSON(t, res.Stdout, &out)
	if out["current-context"] != "default" {
		t.Errorf("current-context = %v", out["current-context"])
	}
	contexts, _ := out["contexts"].(map[string]any)
	def, _ := contexts["default"].(map[string]any)
	if def["token"] != "***" {
		t.Errorf("expected redacted token, got %v", def["token"])
	}
}

func TestConfigViewShowSecrets(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "config", "view", "--show-secrets")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "test-token") {
		t.Fatalf("expected raw token in output: %s", res.Stdout)
	}
}

func TestConfigContextsList(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "config", "contexts")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	var out []map[string]any
	mustDecodeJSON(t, res.Stdout, &out)
	if len(out) != 1 || out[0]["name"] != "default" || out[0]["current"] != true {
		t.Fatalf("got %+v", out)
	}
}

func TestConfigSet(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "config", "set", "base-url=https://new.example.com")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	// Read back via another invocation
	res2 := runCmd(t, "config", "view")
	if !strings.Contains(res2.Stdout, "new.example.com") {
		t.Fatalf("did not persist: %s", res2.Stdout)
	}
}

func TestConfigSetUnknownKey(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "config", "set", "nope=value")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestConfigSetMalformed(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "config", "set", "no-equals-sign")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit=%d", res.ExitCode)
	}
}

func TestConfigUseContextUnknown(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "config", "use-context", "ghost")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestConfigUseContextSwitches(t *testing.T) {
	fb := newFakeBamboo(t)
	// Add a second context via "set", then switch.
	runCmdEnv(t, fb, "auth", "login", "--name", "alt", "--base-url", fb.URL(), "--token", "x")
	res := runCmd(t, "config", "use-context", "alt")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	res2 := runCmd(t, "config", "view")
	var out map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(res2.Stdout)), &out); err != nil {
		t.Fatal(err)
	}
	if out["current-context"] != "alt" {
		t.Fatalf("current-context = %v", out["current-context"])
	}
}
