package api

import (
	"context"
	"strings"
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

// TestBuildHistoryDecodesNestedEntityKey is a regression test for a real-world
// Bamboo response where planResultKey.entityKey is an object {"key":"..."},
// not a bare string.
func TestBuildHistoryDecodesNestedEntityKey(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-PLAN", 200, `{
		"results":{"size":1,"max-result":25,"start-index":0,"result":[
			{"key":"PROJ-PLAN-59","buildNumber":59,"state":"Successful",
			 "planResultKey":{"key":"PROJ-PLAN-59","entityKey":{"key":"PROJ-PLAN"},"resultNumber":59}}
		]}
	}`)
	c := newTestClient(t, fb)
	page, err := c.BuildHistory(context.Background(), "PROJ-PLAN", PageOpts{})
	if err != nil {
		t.Fatalf("BuildHistory: %v", err)
	}
	if len(page.Results) != 1 {
		t.Fatalf("results = %+v", page.Results)
	}
	if got := page.Results[0].PlanResultKey.EntityKey.Key; got != "PROJ-PLAN" {
		t.Errorf("EntityKey.Key = %q, want PROJ-PLAN", got)
	}
	if got := page.Results[0].PlanResultKey.Key; got != "PROJ-PLAN-59" {
		t.Errorf("PlanResultKey.Key = %q", got)
	}
}

func TestListQueueEmptyReturnsNonNilSlice(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/queue", 200, `{"queuedBuilds":{"size":0,"max-result":25,"start-index":0}}`)
	c := newTestClient(t, fb)
	page, err := c.ListQueue(context.Background(), PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Results == nil {
		t.Fatalf("expected non-nil empty slice, got nil")
	}
	if len(page.Results) != 0 {
		t.Fatalf("expected empty, got %+v", page.Results)
	}
}

func TestListQueue(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/queue", 200, `{
		"queuedBuilds":{"size":1,"max-result":25,"start-index":0,
			"queuedBuild":[{"planKey":"PROJ-A","buildNumber":1,"buildResultKey":"PROJ-A-1"}]
		}
	}`)
	c := newTestClient(t, fb)
	page, err := c.ListQueue(context.Background(), PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Results) != 1 || page.Results[0].BuildResultKey != "PROJ-A-1" {
		t.Fatalf("results = %+v", page.Results)
	}
}

func TestTriggerBuildPassesVariables(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("POST", "/rest/api/latest/queue/PROJ-PLAN", 200, `{"planKey":"PROJ-PLAN","buildNumber":42}`)
	c := newTestClient(t, fb)
	qb, err := c.TriggerBuild(context.Background(), "PROJ-PLAN", map[string]string{"BRANCH": "main", "ENV": "prod"})
	if err != nil {
		t.Fatalf("TriggerBuild: %v", err)
	}
	if qb.BuildNumber != 42 {
		t.Errorf("buildNumber = %d", qb.BuildNumber)
	}
	if !strings.Contains(rec.RawQuery, "bamboo.variable.BRANCH=main") {
		t.Errorf("BRANCH query missing: %q", rec.RawQuery)
	}
	if !strings.Contains(rec.RawQuery, "bamboo.variable.ENV=prod") {
		t.Errorf("ENV query missing: %q", rec.RawQuery)
	}
}

func TestStopAndContinueBuild(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	stopRec := fb.expect("DELETE", "/rest/api/latest/queue/PROJ-PLAN-1", 204, "")
	contRec := fb.expect("PUT", "/rest/api/latest/queue/PROJ-PLAN-1", 204, "")
	c := newTestClient(t, fb)
	if err := c.StopBuild(context.Background(), "PROJ-PLAN-1"); err != nil {
		t.Fatalf("StopBuild: %v", err)
	}
	if stopRec.Method != "DELETE" {
		t.Errorf("stop method = %q", stopRec.Method)
	}
	if err := c.ContinueBuild(context.Background(), "PROJ-PLAN-1"); err != nil {
		t.Fatalf("ContinueBuild: %v", err)
	}
	if contRec.Method != "PUT" {
		t.Errorf("continue method = %q", contRec.Method)
	}
}

func TestGetBuildStatus(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/status/PROJ-PLAN-1", 200, `{"finished":false,"progress":{"percentageCompleted":0.4}}`)
	c := newTestClient(t, fb)
	s, err := c.GetBuildStatus(context.Background(), "PROJ-PLAN-1")
	if err != nil {
		t.Fatal(err)
	}
	if s.Finished {
		t.Errorf("Finished = true")
	}
	if s.Progress == nil || s.Progress.PercentageCompleted < 0.39 {
		t.Errorf("progress = %+v", s.Progress)
	}
}

func TestGetBuildAndHistory(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-PLAN-1", 200, `{"key":"PROJ-PLAN-1","buildNumber":1,"state":"Successful"}`)
	fb.expect("GET", "/rest/api/latest/result/PROJ-PLAN", 200, `{
		"results":{"size":2,"max-result":25,"start-index":0,
			"result":[
				{"key":"PROJ-PLAN-2","buildNumber":2,"state":"Successful"},
				{"key":"PROJ-PLAN-1","buildNumber":1,"state":"Failed"}
			]}
	}`)
	c := newTestClient(t, fb)
	if br, err := c.GetBuild(context.Background(), "PROJ-PLAN-1"); err != nil || br.Key != "PROJ-PLAN-1" {
		t.Fatalf("GetBuild: %v %+v", err, br)
	}
	page, err := c.BuildHistory(context.Background(), "PROJ-PLAN", PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Results) != 2 || page.Results[0].BuildNumber != 2 {
		t.Errorf("results = %+v", page.Results)
	}
}

func TestLatestBuilds(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result", 200, `{
		"results":{"size":1,"max-result":25,"start-index":0,"result":[{"key":"X-Y-1"}]}
	}`)
	c := newTestClient(t, fb)
	p, err := c.LatestBuilds(context.Background(), PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Results) != 1 || p.Results[0].Key != "X-Y-1" {
		t.Errorf("results = %+v", p.Results)
	}
}

func TestBuildCommentsCRUD(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	listRec := fb.expect("GET", "/rest/api/latest/result/PROJ-PLAN-1/comment", 200, `{
		"comments":{"size":1,"max-result":25,"start-index":0,
			"comment":[{"id":7,"author":"me","content":"hi"}]}
	}`)
	// Bamboo Server 8.2.4 returns 204 No Content for comment creation.
	addRec := fb.expect("POST", "/rest/api/latest/result/PROJ-PLAN-1/comment", 204, "")
	fb.expect("DELETE", "/rest/api/latest/result/PROJ-PLAN-1/comment/42", 204, "")
	c := newTestClient(t, fb)

	if cs, err := c.ListBuildComments(context.Background(), "PROJ-PLAN-1"); err != nil || len(cs) != 1 || cs[0].ID != 7 {
		t.Fatalf("ListBuildComments: %v %+v", err, cs)
	}
	if !strings.Contains(listRec.RawQuery, "expand=comments.comment") {
		t.Errorf("expected expand=comments.comment, got %q", listRec.RawQuery)
	}
	if err := c.AddBuildComment(context.Background(), "PROJ-PLAN-1", "hi"); err != nil {
		t.Fatalf("AddBuildComment: %v", err)
	}
	if !strings.Contains(string(addRec.Body), `"content":"hi"`) {
		t.Errorf("Add body = %s", addRec.Body)
	}
	if err := c.DeleteBuildComment(context.Background(), "PROJ-PLAN-1", 42); err != nil {
		t.Fatalf("DeleteBuildComment: %v", err)
	}
}

func TestGetBuildLogHappyPath(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1", 200, `{
		"stages":{"stage":[{
			"results":{"result":[
				{"key":"PROJ-A-JOB1-1","buildNumber":1}
			]}
		}]}
	}`)
	logRec := fb.expect("GET", "/download/PROJ-A-JOB1/build_logs/PROJ-A-JOB1-1.log", 200, "compile output\nfinished\n")
	c := newTestClient(t, fb)

	logs, err := c.GetBuildLog(context.Background(), "PROJ-A-1")
	if err != nil {
		t.Fatalf("GetBuildLog: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 job log, got %+v", logs)
	}
	if logs[0].JobKey != "PROJ-A-JOB1-1" {
		t.Errorf("JobKey = %q", logs[0].JobKey)
	}
	if !strings.Contains(logs[0].Log, "compile output") {
		t.Errorf("Log content = %q", logs[0].Log)
	}
	if logRec.AuthHeader != "Bearer test-token" {
		t.Errorf("download must still send Bearer auth: %q", logRec.AuthHeader)
	}
}

func TestGetBuildLogMultiJob(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1", 200, `{
		"stages":{"stage":[
			{"results":{"result":[{"key":"PROJ-A-JOB1-1","buildNumber":1}]}},
			{"results":{"result":[{"key":"PROJ-A-JOB2-1","buildNumber":1}]}}
		]}
	}`)
	fb.expect("GET", "/download/PROJ-A-JOB1/build_logs/PROJ-A-JOB1-1.log", 200, "job1 log\n")
	fb.expect("GET", "/download/PROJ-A-JOB2/build_logs/PROJ-A-JOB2-1.log", 200, "job2 log\n")
	c := newTestClient(t, fb)

	logs, err := c.GetBuildLog(context.Background(), "PROJ-A-1")
	if err != nil {
		t.Fatalf("GetBuildLog: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 job logs, got %d", len(logs))
	}
	if logs[0].Log != "job1 log\n" || logs[1].Log != "job2 log\n" {
		t.Errorf("logs = %+v", logs)
	}
}

func TestGetBuildLogDetectsHTMLLoginRedirect(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1", 200, `{
		"stages":{"stage":[{"results":{"result":[{"key":"PROJ-A-JOB1-1","buildNumber":1}]}}]}
	}`)
	// Bamboo's session-auth redirect: HTTP 200 with an HTML login page.
	fb.expect("GET", "/download/PROJ-A-JOB1/build_logs/PROJ-A-JOB1-1.log", 200,
		`<!DOCTYPE html><html><head><title>Log in to Bamboo</title></head>...`)
	c := newTestClient(t, fb)

	_, err := c.GetBuildLog(context.Background(), "PROJ-A-1")
	if err == nil {
		t.Fatal("expected session_auth_required error, got nil")
	}
	var fe *fail.Error
	if !errAs(err, &fe) {
		t.Fatalf("expected *fail.Error, got %T %v", err, err)
	}
	if fe.Code != "session_auth_required" {
		t.Errorf("Code = %q", fe.Code)
	}
	if fe.Exit != fail.ExitAuth {
		t.Errorf("Exit = %d, want %d", fe.Exit, fail.ExitAuth)
	}
}

func TestGetBuildLogEmptyStagesReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-A-1", 200, `{"stages":{"stage":[]}}`)
	c := newTestClient(t, fb)
	logs, err := c.GetBuildLog(context.Background(), "PROJ-A-1")
	if err != nil {
		t.Fatalf("GetBuildLog: %v", err)
	}
	if logs == nil || len(logs) != 0 {
		t.Fatalf("expected non-nil empty slice, got %+v", logs)
	}
}

func TestBuildLabelsCRUD(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/result/PROJ-PLAN-1/label", 200, `{
		"labels":{"size":1,"max-result":25,"start-index":0,"label":[{"name":"green"}]}
	}`)
	addRec := fb.expect("POST", "/rest/api/latest/result/PROJ-PLAN-1/label", 200, "")
	fb.expect("DELETE", "/rest/api/latest/result/PROJ-PLAN-1/label/green", 204, "")
	c := newTestClient(t, fb)

	if ls, err := c.ListBuildLabels(context.Background(), "PROJ-PLAN-1"); err != nil || len(ls) != 1 || ls[0].Name != "green" {
		t.Fatalf("ListBuildLabels: %v %+v", err, ls)
	}
	if err := c.AddBuildLabel(context.Background(), "PROJ-PLAN-1", "red"); err != nil {
		t.Fatalf("AddBuildLabel: %v", err)
	}
	if !strings.Contains(string(addRec.Body), `"name":"red"`) {
		t.Errorf("Add body = %s", addRec.Body)
	}
	if err := c.DeleteBuildLabel(context.Background(), "PROJ-PLAN-1", "green"); err != nil {
		t.Fatalf("DeleteBuildLabel: %v", err)
	}
}
