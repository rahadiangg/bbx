package cmd

import (
	"strings"
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

func TestPlanList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan", 200, `{
		"plans":{"size":1,"max-result":25,"start-index":0,
			"plan":[{"key":"PROJ-A","name":"Alpha","enabled":true,"projectName":"P"}]
		}
	}`)
	res := runCmdEnv(t, fb, "plan", "list")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	var out []map[string]any
	mustDecodeJSON(t, res.Stdout, &out)
	if len(out) != 1 || out[0]["key"] != "PROJ-A" {
		t.Fatalf("got %+v", out)
	}
}

func TestPlanListAllPaginates(t *testing.T) {
	fb := newFakeBamboo(t)
	// Use a single route; the harness regenerates Capture per route, but the
	// stub returns the same payload for both pages. We just want to verify the
	// CLI invokes paginate without errors.
	fb.expect("GET", "/rest/api/latest/plan", 200, `{
		"plans":{"size":0,"max-result":25,"start-index":0,"plan":[]}
	}`)
	res := runCmdEnv(t, fb, "plan", "list", "--all")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestPlanGet(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A", 200, `{"key":"PROJ-A","name":"Alpha","enabled":true}`)
	res := runCmdEnv(t, fb, "plan", "get", "PROJ-A")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	var p map[string]any
	mustDecodeJSON(t, res.Stdout, &p)
	if p["key"] != "PROJ-A" {
		t.Fatalf("got %+v", p)
	}
}

func TestPlanGetNotFound(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/MISSING", 404, `{"message":"Plan not found"}`)
	res := runCmdEnv(t, fb, "plan", "get", "MISSING")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestPlanEnableDisable(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("POST", "/rest/api/latest/plan/PROJ-A/enable", 204, "")
	fb.expect("DELETE", "/rest/api/latest/plan/PROJ-A/enable", 204, "")
	if r := runCmdEnv(t, fb, "plan", "enable", "PROJ-A"); r.ExitCode != fail.ExitOK {
		t.Fatalf("enable: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "plan", "disable", "PROJ-A"); r.ExitCode != fail.ExitOK {
		t.Fatalf("disable: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
}

func TestPlanDeleteRequiresYes(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("DELETE", "/rest/api/latest/plan/PROJ-A", 204, "")
	res := runCmdEnv(t, fb, "plan", "delete", "PROJ-A")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("expected confirmation_required, exit=%d", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "confirmation_required") {
		t.Fatalf("stderr=%s", res.Stderr)
	}
	res2 := runCmd(t, "plan", "delete", "PROJ-A", "--yes")
	if res2.ExitCode != fail.ExitOK {
		t.Fatalf("with --yes: exit=%d stderr=%s", res2.ExitCode, res2.Stderr)
	}
}

func TestPlanBranchTree(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/branch", 200, `{
		"branches":{"size":1,"max-result":25,"start-index":0,
			"branch":[{"key":"PROJ-A0","name":"feature/x","enabled":true}]}
	}`)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/branch/feature%2Fx", 200, `{"key":"k","name":"feature/x","enabled":true}`)
	fb.expect("PUT", "/rest/api/latest/plan/PROJ-A/branch/feature%2Fx", 200, `{"key":"k","name":"feature/x"}`)

	if r := runCmdEnv(t, fb, "plan", "branch", "list", "PROJ-A"); r.ExitCode != fail.ExitOK {
		t.Fatalf("list: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "plan", "branch", "get", "PROJ-A", "feature/x"); r.ExitCode != fail.ExitOK {
		t.Fatalf("get: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "plan", "branch", "create", "PROJ-A", "feature/x", "--vcs-branch", "main"); r.ExitCode != fail.ExitOK {
		t.Fatalf("create: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
}

func TestPlanVariableTree(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/variables", 200, `[{"name":"K","value":"v"}]`)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/variables/K", 200, `{"name":"K","value":"v"}`)
	fb.expect("PUT", "/rest/api/latest/plan/PROJ-A/variables/K", 200, `{"name":"K","value":"v2"}`)
	fb.expect("DELETE", "/rest/api/latest/plan/PROJ-A/variables/K", 204, "")

	if r := runCmdEnv(t, fb, "plan", "variable", "list", "PROJ-A"); r.ExitCode != fail.ExitOK {
		t.Fatalf("list: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "plan", "variable", "get", "PROJ-A", "K"); r.ExitCode != fail.ExitOK {
		t.Fatalf("get: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "plan", "variable", "set", "PROJ-A", "K", "v2"); r.ExitCode != fail.ExitOK {
		t.Fatalf("set: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "plan", "variable", "delete", "PROJ-A", "K"); r.ExitCode != fail.ExitOK {
		t.Fatalf("delete: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
}

func TestPlanClone(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("PUT", "/rest/api/latest/clone/RES-TES:RES-BBX", 200,
		`{"key":"RES-BBX","name":"research-project - testttt","enabled":false}`)
	res := runCmdEnv(t, fb, "plan", "clone", "RES-TES", "RES-BBX")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "RES-BBX") {
		t.Fatalf("stdout missing destination key: %s", res.Stdout)
	}
}

func TestPlanCloneRequiresTwoArgs(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "plan", "clone", "ONLY-ONE")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit=%d, want %d", res.ExitCode, fail.ExitUsage)
	}
}

func TestPlanBranchEnableDisable(t *testing.T) {
	fb := newFakeBamboo(t)
	// Both enable + disable do: GET /branch/{name} -> POST or DELETE /plan/{branchKey}/enable
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/branch/feature%2Fx", 200, `{"key":"PROJ-A0","name":"feature/x"}`)
	fb.expect("POST", "/rest/api/latest/plan/PROJ-A0/enable", 204, "")

	if r := runCmdEnv(t, fb, "plan", "branch", "enable", "PROJ-A", "feature/x"); r.ExitCode != fail.ExitOK {
		t.Fatalf("enable: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}

	// Re-register the GET route for disable (each route is consumed per-call only
	// via the shared mux but our harness allows repeated calls). Add DELETE.
	fb.expect("DELETE", "/rest/api/latest/plan/PROJ-A0/enable", 204, "")
	if r := runCmd(t, "plan", "branch", "disable", "PROJ-A", "feature/x"); r.ExitCode != fail.ExitOK {
		t.Fatalf("disable: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
}

func TestPlanBranchDelete(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/branch/feature%2Fx", 200, `{"key":"PROJ-A0","name":"feature/x"}`)
	fb.expect("DELETE", "/rest/api/latest/plan/PROJ-A0", 204, "")
	res := runCmdEnv(t, fb, "plan", "branch", "delete", "PROJ-A", "feature/x")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestPlanVariableSetFallsBackToCreateOn404(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("PUT", "/rest/api/latest/plan/PROJ-A/variables/NEW", 404, `{"message":"not found"}`)
	fb.expect("POST", "/rest/api/latest/plan/PROJ-A/variables", 201, `{"name":"NEW","value":"v"}`)
	res := runCmdEnv(t, fb, "plan", "variable", "set", "PROJ-A", "NEW", "v")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}
