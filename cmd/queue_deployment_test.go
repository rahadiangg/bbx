package cmd

import (
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

func TestQueueList(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/queue", 200, `{"queuedBuilds":{"size":0,"max-result":25,"start-index":0,"queuedBuild":[]}}`)
	res := runCmdEnv(t, fb, "queue", "list")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentQueue(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/queue/deployment", 200, `{
		"queuedDeployments":{"size":0,"start-index":0,"max-result":0},
		"inProgress":{"size":0,"start-index":0,"max-result":0}
	}`)
	res := runCmdEnv(t, fb, "deployment", "queue")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentTriggerRequiresFlags(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "deployment", "trigger")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentTrigger(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("POST", "/rest/api/latest/queue/deployment", 200, `{"id":42,"deploymentState":"PENDING"}`)
	res := runCmdEnv(t, fb, "deployment", "trigger", "--environment-id", "7", "--version-id", "9")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestDeploymentCancelResultPreview(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("DELETE", "/rest/api/latest/queue/deployment/42", 204, "")
	fb.expect("GET", "/rest/api/latest/deploy/result/42", 200, `{"id":42,"deploymentState":"SUCCEEDED"}`)
	fb.expect("GET", "/rest/api/latest/deploy/preview/version", 200, `{"nextVersionName":"v2"}`)

	if r := runCmdEnv(t, fb, "deployment", "cancel", "42"); r.ExitCode != fail.ExitOK {
		t.Fatalf("cancel: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "deployment", "result", "42"); r.ExitCode != fail.ExitOK {
		t.Fatalf("result: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "deployment", "preview", "--project-id", "5"); r.ExitCode != fail.ExitOK {
		t.Fatalf("preview: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
}

func TestDeploymentBadID(t *testing.T) {
	fb := newFakeBamboo(t)
	if r := runCmdEnv(t, fb, "deployment", "cancel", "abc"); r.ExitCode != fail.ExitUsage {
		t.Fatalf("cancel: exit=%d", r.ExitCode)
	}
	if r := runCmd(t, "deployment", "result", "xyz"); r.ExitCode != fail.ExitUsage {
		t.Fatalf("result: exit=%d", r.ExitCode)
	}
}
