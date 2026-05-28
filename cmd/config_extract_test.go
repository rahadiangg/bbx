package cmd

import (
	"strings"
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

// Tests covering the new config-extraction surface:
//   bbx plan spec/config/artifact/vcs-branches
//   bbx project get/spec/variable/repository
//   bbx deployment project list/get/spec/repository
//   bbx deployment environment get/variable/requirement/agent
//   bbx deployment version list

// --- plan side ---

func TestPlanSpecRawJava(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/specs", 200,
		`{"spec":{"projectKey":"PROJ","buildKey":"A","code":"package PROJ;\n@BambooSpec\npublic class A {}\n"}}`)
	res := runCmdEnv(t, fb, "plan", "spec", "PROJ-A", "-o", "json")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "@BambooSpec") {
		t.Fatalf("stdout missing Java content: %s", res.Stdout)
	}
}

func TestPlanConfig(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A", 200,
		`{"key":"PROJ-A","stages":{"size":1,"stage":[{"name":"Default"}]}}`)
	res := runCmdEnv(t, fb, "plan", "config", "PROJ-A")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "PROJ-A") || !strings.Contains(res.Stdout, "Default") {
		t.Fatalf("stdout = %s", res.Stdout)
	}
}

func TestPlanArtifactList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/artifact", 200,
		`{"artifacts":{"size":1,"max-result":25,"start-index":0,"artifact":[{"name":"binaries","location":"out/"}]}}`)
	res := runCmdEnv(t, fb, "plan", "artifact", "list", "PROJ-A")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "binaries") {
		t.Fatalf("stdout = %s", res.Stdout)
	}
}

func TestPlanVCSBranches(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/vcsBranches", 200,
		`{"branches":{"size":2,"max-result":25,"start-index":0,"branch":[{"name":"main"},{"name":"dev"}]}}`)
	res := runCmdEnv(t, fb, "plan", "vcs-branches", "PROJ-A")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "main") || !strings.Contains(res.Stdout, "dev") {
		t.Fatalf("stdout = %s", res.Stdout)
	}
}

// --- project side ---

func TestProjectGet(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/ANVIS", 200, `{"key":"ANVIS","name":"Anvis"}`)
	res := runCmdEnv(t, fb, "project", "get", "ANVIS")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "Anvis") {
		t.Fatalf("stdout = %s", res.Stdout)
	}
}

func TestProjectSpec(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/PROJ/specs", 200,
		`{"projectKey":"PROJ","spec":[{"projectKey":"PROJ","buildKey":"A","code":"// A"}]}`)
	res := runCmdEnv(t, fb, "project", "spec", "PROJ", "-o", "json")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "PROJ") {
		t.Fatalf("stdout = %s", res.Stdout)
	}
}

func TestProjectVariableList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/ANVIS/variables", 200, `[{"name":"K","value":"v"}]`)
	res := runCmdEnv(t, fb, "project", "variable", "list", "ANVIS")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, `"K"`) {
		t.Fatalf("stdout = %s", res.Stdout)
	}
}

func TestProjectVariableGet(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/ANVIS/variable/K", 200, `{"name":"K","value":"v"}`)
	res := runCmdEnv(t, fb, "project", "variable", "get", "ANVIS", "K")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestProjectRepositoryList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/ANVIS/repository", 200, `[{"id":1,"name":"r1"}]`)
	res := runCmdEnv(t, fb, "project", "repository", "list", "ANVIS")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

// --- deployment project ---

func TestDeploymentProjectList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/all", 200,
		`[{"id":1,"name":"DP1"},{"id":2,"name":"DP2"}]`)
	res := runCmdEnv(t, fb, "deployment", "project", "list")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentProjectListForPlan(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/forPlan", 200, `[]`)
	res := runCmdEnv(t, fb, "deployment", "project", "list", "--for-plan", "PROJ-A")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentProjectGet(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/42", 200,
		`{"id":42,"name":"Prod"}`)
	res := runCmdEnv(t, fb, "deployment", "project", "get", "42")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentProjectGetInvalidID(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "deployment", "project", "get", "abc")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("expected ExitUsage, got %d", res.ExitCode)
	}
}

func TestDeploymentProjectSpec(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/42/specs", 200,
		`{"deploymentId":42,"code":"import com.atlassian.bamboo.specs..."}`)
	res := runCmdEnv(t, fb, "deployment", "project", "spec", "42", "-o", "json")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentProjectRepositoryList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/42/repository", 200, `[]`)
	res := runCmdEnv(t, fb, "deployment", "project", "repository", "list", "42")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

// --- deployment environment ---

func TestDeploymentEnvironmentGet(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/environment/100", 200,
		`{"id":100,"name":"prod"}`)
	res := runCmdEnv(t, fb, "deployment", "environment", "get", "100")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentEnvironmentVariableList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/environment/100/variables", 200, `[]`)
	res := runCmdEnv(t, fb, "deployment", "environment", "variable", "list", "100")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentEnvironmentRequirementList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/environment/100/requirement", 200, `[]`)
	res := runCmdEnv(t, fb, "deployment", "environment", "requirement", "list", "100")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentEnvironmentAgentList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/environment/100/agent-assignment", 200, `[]`)
	res := runCmdEnv(t, fb, "deployment", "environment", "agent", "list", "100")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentEnvironmentInvalidID(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "deployment", "environment", "get", "abc")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("expected ExitUsage, got %d", res.ExitCode)
	}
}

// --- deployment versions ---

func TestDeploymentVersionList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/42/versions", 200,
		`{"size":1,"versions":[{"id":1,"name":"v1"}]}`)
	res := runCmdEnv(t, fb, "deployment", "version", "list", "42")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "v1") {
		t.Fatalf("stdout = %s", res.Stdout)
	}
}
