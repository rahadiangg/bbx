package cmd

import (
	"strings"
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

func TestBuildTrigger(t *testing.T) {
	fb := newFakeBamboo(t)
	rec := fb.expect("POST", "/rest/api/latest/queue/PROJ-A", 200, `{"planKey":"PROJ-A","buildNumber":7,"buildResultKey":"PROJ-A-7"}`)
	res := runCmdEnv(t, fb, "build", "trigger", "PROJ-A", "--var", "FOO=bar")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(rec.RawQuery, "bamboo.variable.FOO=bar") {
		t.Fatalf("query = %q", rec.RawQuery)
	}
	if !strings.Contains(res.Stdout, "PROJ-A-7") {
		t.Fatalf("stdout missing key: %s", res.Stdout)
	}
}

func TestBuildTriggerBadVar(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "build", "trigger", "PROJ-A", "--var", "no-equals")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit=%d", res.ExitCode)
	}
}

func TestBuildStopContinue(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("DELETE", "/rest/api/latest/queue/PROJ-A-1", 204, "")
	fb.expect("PUT", "/rest/api/latest/queue/PROJ-A-1", 204, "")
	if r := runCmdEnv(t, fb, "build", "stop", "PROJ-A-1"); r.ExitCode != fail.ExitOK {
		t.Fatalf("stop: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "build", "continue", "PROJ-A-1"); r.ExitCode != fail.ExitOK {
		t.Fatalf("continue: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
}

func TestBuildStatusGetHistoryLatest(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/status/PROJ-A-1", 200, `{"finished":true}`)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1", 200, `{"key":"PROJ-A-1","state":"Successful"}`)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A", 200, `{"results":{"size":0,"max-result":25,"start-index":0,"result":[]}}`)
	fb.expect("GET", "/rest/api/latest/result", 200, `{"results":{"size":0,"max-result":25,"start-index":0,"result":[]}}`)

	for _, args := range [][]string{
		{"build", "status", "PROJ-A-1"},
		{"build", "get", "PROJ-A-1"},
		{"build", "history", "PROJ-A"},
		{"build", "latest"},
	} {
		args := args
		r := runCmdEnv(t, fb, args...)
		if r.ExitCode != fail.ExitOK {
			t.Errorf("%v: exit=%d stderr=%s", args, r.ExitCode, r.Stderr)
		}
	}
}

func TestBuildCommentTree(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1/comment", 200, `{"comments":{"size":0,"start-index":0,"max-result":0,"comment":[]}}`)
	fb.expect("POST", "/rest/api/latest/result/PROJ-A-1/comment", 200, `{"id":1,"content":"hi"}`)
	fb.expect("DELETE", "/rest/api/latest/result/PROJ-A-1/comment/1", 204, "")

	if r := runCmdEnv(t, fb, "build", "comment", "list", "PROJ-A-1"); r.ExitCode != fail.ExitOK {
		t.Fatalf("list: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "build", "comment", "add", "PROJ-A-1", "hi"); r.ExitCode != fail.ExitOK {
		t.Fatalf("add: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "build", "comment", "delete", "PROJ-A-1", "1"); r.ExitCode != fail.ExitOK {
		t.Fatalf("delete: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
}

func TestBuildCommentDeleteBadID(t *testing.T) {
	fb := newFakeBamboo(t)
	res := runCmdEnv(t, fb, "build", "comment", "delete", "PROJ-A-1", "abc")
	if res.ExitCode != fail.ExitUsage {
		t.Fatalf("exit=%d", res.ExitCode)
	}
}

func TestBuildLogHappyPath(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1", 200,
		`{"stages":{"stage":[{"results":{"result":[{"key":"PROJ-A-JOB1-1","buildNumber":1}]}}]}}`)
	fb.expect("GET", "/download/PROJ-A-JOB1/build_logs/PROJ-A-JOB1-1.log", 200, "hello-from-bamboo\n")
	res := runCmdEnv(t, fb, "build", "log", "PROJ-A-1")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "hello-from-bamboo") {
		t.Fatalf("stdout missing log content: %q", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "PROJ-A-JOB1-1") {
		t.Fatalf("stdout missing job key marker: %q", res.Stdout)
	}
}

func TestBuildLogSessionAuthRequired(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1", 200,
		`{"stages":{"stage":[{"results":{"result":[{"key":"PROJ-A-JOB1-1","buildNumber":1}]}}]}}`)
	fb.expect("GET", "/download/PROJ-A-JOB1/build_logs/PROJ-A-JOB1-1.log", 200,
		`<!DOCTYPE html><html><head><title>Log in</title></head>`)
	res := runCmdEnv(t, fb, "build", "log", "PROJ-A-1")
	if res.ExitCode != fail.ExitAuth {
		t.Fatalf("exit=%d, want %d (auth)\nstderr=%s", res.ExitCode, fail.ExitAuth, res.Stderr)
	}
	if !strings.Contains(res.Stderr, "session_auth_required") {
		t.Fatalf("stderr should mention session_auth_required: %q", res.Stderr)
	}
}

func TestBuildLogJobFilter(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1", 200, `{
		"stages":{"stage":[
			{"results":{"result":[{"key":"PROJ-A-JOB1-1","buildNumber":1}]}},
			{"results":{"result":[{"key":"PROJ-A-JOB2-1","buildNumber":1}]}}
		]}
	}`)
	fb.expect("GET", "/download/PROJ-A-JOB1/build_logs/PROJ-A-JOB1-1.log", 200, "job1 log\n")
	fb.expect("GET", "/download/PROJ-A-JOB2/build_logs/PROJ-A-JOB2-1.log", 200, "job2 log\n")
	res := runCmdEnv(t, fb, "build", "log", "PROJ-A-1", "--job", "PROJ-A-JOB2-1")
	if res.ExitCode != fail.ExitOK {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if strings.Contains(res.Stdout, "job1 log") {
		t.Errorf("--job filter should have excluded job1: %s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "job2 log") {
		t.Errorf("--job filter missing job2: %s", res.Stdout)
	}
}

func TestBuildLabelTree(t *testing.T) {
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1/label", 200, `{"labels":{"size":0,"start-index":0,"max-result":0,"label":[]}}`)
	fb.expect("POST", "/rest/api/latest/result/PROJ-A-1/label", 204, "")
	fb.expect("DELETE", "/rest/api/latest/result/PROJ-A-1/label/red", 204, "")

	if r := runCmdEnv(t, fb, "build", "label", "list", "PROJ-A-1"); r.ExitCode != fail.ExitOK {
		t.Fatalf("list: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "build", "label", "add", "PROJ-A-1", "red"); r.ExitCode != fail.ExitOK {
		t.Fatalf("add: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
	if r := runCmd(t, "build", "label", "delete", "PROJ-A-1", "red"); r.ExitCode != fail.ExitOK {
		t.Fatalf("delete: exit=%d stderr=%s", r.ExitCode, r.Stderr)
	}
}
