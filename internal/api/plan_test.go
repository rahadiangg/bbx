package api

import (
	"context"
	"strings"
	"testing"
)

func TestListPlans(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("GET", "/rest/api/latest/plan", 200, `{
		"plans":{
			"size":2,"max-result":25,"start-index":0,
			"plan":[
				{"key":"PROJ-A","name":"Alpha","enabled":true,"projectKey":"PROJ","projectName":"Proj"},
				{"key":"PROJ-B","name":"Beta","enabled":false,"projectKey":"PROJ"}
			]
		}
	}`)
	c := newTestClient(t, fb)

	page, err := c.ListPlans(context.Background(), PageOpts{MaxResults: 25})
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if !strings.Contains(rec.RawQuery, "expand=plans") {
		t.Errorf("expected default expand=plans, got %q", rec.RawQuery)
	}
	if !strings.Contains(rec.RawQuery, "max-results=25") {
		t.Errorf("expected max-results=25, got %q", rec.RawQuery)
	}
	if len(page.Results) != 2 || page.Size != 2 {
		t.Fatalf("unexpected page: %+v", page)
	}
	if page.Results[0].Key != "PROJ-A" || page.Results[1].Name != "Beta" {
		t.Errorf("results = %+v", page.Results)
	}
}

func TestListPlansHonorsExpandOverride(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("GET", "/rest/api/latest/plan", 200, `{"plans":{"size":0,"max-result":0,"start-index":0,"plan":[]}}`)
	c := newTestClient(t, fb)
	_, err := c.ListPlans(context.Background(), PageOpts{Expand: "plans.plan"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rec.RawQuery, "expand=plans.plan") {
		t.Fatalf("expand override lost: %q", rec.RawQuery)
	}
}

func TestGetPlan(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-PLAN", 200, `{"key":"PROJ-PLAN","name":"Plan","enabled":true}`)
	c := newTestClient(t, fb)

	p, err := c.GetPlan(context.Background(), "PROJ-PLAN")
	if err != nil {
		t.Fatal(err)
	}
	if p.Key != "PROJ-PLAN" || p.Name != "Plan" || !p.Enabled {
		t.Errorf("plan = %+v", p)
	}
}

func TestEnableDisableDeletePlan(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	en := fb.expect("POST", "/rest/api/latest/plan/PROJ-PLAN/enable", 204, "")
	di := fb.expect("DELETE", "/rest/api/latest/plan/PROJ-PLAN/enable", 204, "")
	de := fb.expect("DELETE", "/rest/api/latest/plan/PROJ-PLAN", 204, "")
	c := newTestClient(t, fb)

	if err := c.EnablePlan(context.Background(), "PROJ-PLAN"); err != nil {
		t.Fatalf("EnablePlan: %v", err)
	}
	if en.Method != "POST" {
		t.Errorf("enable method = %q", en.Method)
	}
	if err := c.DisablePlan(context.Background(), "PROJ-PLAN"); err != nil {
		t.Fatalf("DisablePlan: %v", err)
	}
	if di.Method != "DELETE" {
		t.Errorf("disable method = %q", di.Method)
	}
	if err := c.DeletePlan(context.Background(), "PROJ-PLAN"); err != nil {
		t.Fatalf("DeletePlan: %v", err)
	}
	if de.Method != "DELETE" {
		t.Errorf("delete method = %q", de.Method)
	}
}

func TestListPlanBranches(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("GET", "/rest/api/latest/plan/PROJ-PLAN/branch", 200, `{
		"branches":{"size":1,"max-result":25,"start-index":0,"branch":[
			{"key":"PROJ-PLAN0","name":"feature/x","enabled":true}
		]}
	}`)
	c := newTestClient(t, fb)
	page, err := c.ListPlanBranches(context.Background(), "PROJ-PLAN", PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rec.RawQuery, "expand=branches") {
		t.Errorf("default expand lost: %q", rec.RawQuery)
	}
	if len(page.Results) != 1 || page.Results[0].Name != "feature/x" {
		t.Fatalf("results = %+v", page.Results)
	}
}

func TestGetPlanBranch(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-PLAN/branch/feature%2Fx", 200, `{"key":"k","name":"feature/x","enabled":true}`)
	c := newTestClient(t, fb)
	got, err := c.GetPlanBranch(context.Background(), "PROJ-PLAN", "feature/x")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "feature/x" {
		t.Errorf("got %+v", got)
	}
}

func TestCreatePlanBranch(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("PUT", "/rest/api/latest/plan/PROJ-PLAN/branch/feature%2Fx", 200, `{"key":"k","name":"feature/x"}`)
	c := newTestClient(t, fb)
	if _, err := c.CreatePlanBranch(context.Background(), "PROJ-PLAN", "feature/x", "vcs-branch"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rec.RawQuery, "vcsBranch=vcs-branch") {
		t.Errorf("vcsBranch missing: %q", rec.RawQuery)
	}
}

func TestPlanVariablesCRUD(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	// Real Bamboo returns variables as a top-level array with {name,value} keys.
	fb.expect("GET", "/rest/api/latest/plan/PROJ-PLAN/variables", 200, `[{"name":"A","value":"1"}]`)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-PLAN/variables/A", 200, `{"name":"A","value":"1"}`)
	// Bamboo expects name+value as query params on POST, not a JSON body.
	addRec := fb.expect("POST", "/rest/api/latest/plan/PROJ-PLAN/variables", 201, `{"name":"NEW","value":"v"}`)
	putRec := fb.expect("PUT", "/rest/api/latest/plan/PROJ-PLAN/variables/A", 200, `{"name":"A","value":"2"}`)
	fb.expect("DELETE", "/rest/api/latest/plan/PROJ-PLAN/variables/A", 204, "")
	c := newTestClient(t, fb)

	if vs, err := c.ListPlanVariables(context.Background(), "PROJ-PLAN"); err != nil || len(vs) != 1 || vs[0].Name != "A" {
		t.Fatalf("ListPlanVariables: %v %+v", err, vs)
	}
	if v, err := c.GetPlanVariable(context.Background(), "PROJ-PLAN", "A"); err != nil || v.Name != "A" {
		t.Fatalf("GetPlanVariable: %v %+v", err, v)
	}
	if _, err := c.AddPlanVariable(context.Background(), "PROJ-PLAN", "NEW", "v"); err != nil {
		t.Fatalf("AddPlanVariable: %v", err)
	}
	// POST should send the data via query string, not body.
	if !strings.Contains(addRec.RawQuery, "name=NEW") || !strings.Contains(addRec.RawQuery, "value=v") {
		t.Errorf("Add query = %q (expected name=NEW & value=v)", addRec.RawQuery)
	}
	if len(addRec.Body) != 0 {
		t.Errorf("Add body should be empty, got %q", addRec.Body)
	}
	if _, err := c.UpdatePlanVariable(context.Background(), "PROJ-PLAN", "A", "2"); err != nil {
		t.Fatalf("UpdatePlanVariable: %v", err)
	}
	if !strings.Contains(string(putRec.Body), `"value":"2"`) {
		t.Errorf("Put body = %s", putRec.Body)
	}
	if err := c.DeletePlanVariable(context.Background(), "PROJ-PLAN", "A"); err != nil {
		t.Fatalf("DeletePlanVariable: %v", err)
	}
}

// TestClonePlan exercises the happy path and asserts that the colon between
// src and dst remains a literal `:` (not url-encoded as `%3A`) — Bamboo's
// router only matches the literal form.
func TestClonePlan(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("PUT", "/rest/api/latest/clone/RES-TES:RES-BBX", 200,
		`{"key":"RES-BBX","name":"research-project - testttt","enabled":false,"projectKey":"RES","projectName":"research-project"}`)
	c := newTestClient(t, fb)

	got, err := c.ClonePlan(context.Background(), "RES-TES", "RES-BBX")
	if err != nil {
		t.Fatalf("ClonePlan: %v", err)
	}
	if rec.Method != "PUT" {
		t.Errorf("method = %q", rec.Method)
	}
	p, ok := got.(*Plan)
	if !ok {
		t.Fatalf("expected *Plan, got %T", got)
	}
	if p.Key != "RES-BBX" || p.ProjectKey != "RES" {
		t.Fatalf("decoded plan = %+v", p)
	}
}

func TestClonePlanFallbackOnUnknownResponse(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	// Some Bamboo versions return a slim envelope without a usable key field.
	fb.expect("PUT", "/rest/api/latest/clone/RES-TES:RES-BBX", 200, `{"messages":["cloned"]}`)
	c := newTestClient(t, fb)

	got, err := c.ClonePlan(context.Background(), "RES-TES", "RES-BBX")
	if err != nil {
		t.Fatalf("ClonePlan: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map fallback, got %T", got)
	}
	if _, present := m["messages"]; !present {
		t.Fatalf("fallback map missing original payload: %+v", m)
	}
}

func TestClonePlanFallbackOn204(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("PUT", "/rest/api/latest/clone/RES-TES:RES-BBX", 204, "")
	c := newTestClient(t, fb)

	got, err := c.ClonePlan(context.Background(), "RES-TES", "RES-BBX")
	if err != nil {
		t.Fatalf("ClonePlan: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok || m["key"] != "RES-BBX" {
		t.Fatalf("expected synthetic {key:RES-BBX}, got %+v", got)
	}
}

func TestClonePlanRejectsEmptyArgs(t *testing.T) {
	t.Parallel()
	c, _ := New(Options{BaseURL: "http://x", Token: "t"})
	if _, err := c.ClonePlan(context.Background(), "", "DST"); err == nil {
		t.Error("expected error for empty src")
	}
	if _, err := c.ClonePlan(context.Background(), "SRC", ""); err == nil {
		t.Error("expected error for empty dst")
	}
}

// TestDeletePlanBranch verifies the two-step delete flow:
//  1. GET branch to look up its plan key (Bamboo assigns auto keys like PROJ-A0)
//  2. DELETE that plan key
//
// Bamboo 8.2.4 returns 405 on DELETE /plan/{key}/branch/{name}.
func TestDeletePlanBranch(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/branch/feature%2Fx", 200,
		`{"key":"PROJ-A0","name":"feature/x","enabled":true}`)
	delRec := fb.expect("DELETE", "/rest/api/latest/plan/PROJ-A0", 204, "")
	c := newTestClient(t, fb)
	if err := c.DeletePlanBranch(context.Background(), "PROJ-A", "feature/x"); err != nil {
		t.Fatalf("DeletePlanBranch: %v", err)
	}
	if delRec.Method != "DELETE" {
		t.Errorf("expected DELETE on PROJ-A0, got %q", delRec.Method)
	}
}

func TestGetPlanSpec(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/specs", 200,
		`{"spec":{"projectKey":"PROJ","buildKey":"A","code":"package PROJ;\n\n@BambooSpec\npublic class A {}\n"}}`)
	c := newTestClient(t, fb)

	sp, err := c.GetPlanSpec(context.Background(), "PROJ-A")
	if err != nil {
		t.Fatalf("GetPlanSpec: %v", err)
	}
	if sp.ProjectKey != "PROJ" || sp.BuildKey != "A" || sp.Code == "" {
		t.Fatalf("spec = %+v", sp)
	}
	if !strings.Contains(sp.Code, "@BambooSpec") {
		t.Errorf("Code should be Java; got %q", sp.Code)
	}
}

func TestGetPlanConfig(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("GET", "/rest/api/latest/plan/PROJ-A", 200,
		`{"key":"PROJ-A","stages":{"size":1,"stage":[{"name":"Default Stage","plans":{"plan":[{"key":"PROJ-A-JOB1"}]}}]}}`)
	c := newTestClient(t, fb)

	cfg, err := c.GetPlanConfig(context.Background(), "PROJ-A")
	if err != nil {
		t.Fatalf("GetPlanConfig: %v", err)
	}
	if !strings.Contains(rec.RawQuery, "expand=stages.stage.plans.plan") {
		t.Errorf("expected expand query param, got %q", rec.RawQuery)
	}
	if cfg["key"] != "PROJ-A" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestListPlanArtifacts(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/artifact", 200, `{
		"artifacts":{"size":1,"max-result":25,"start-index":0,
			"artifact":[{"name":"binaries","location":"out/","copyPattern":"*.jar","shared":true,"required":false}]
		}
	}`)
	c := newTestClient(t, fb)
	page, err := c.ListPlanArtifacts(context.Background(), "PROJ-A", PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Results) != 1 || page.Results[0].Name != "binaries" || !page.Results[0].Shared {
		t.Errorf("results = %+v", page.Results)
	}
}

func TestListPlanArtifactsEmpty(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/artifact", 200,
		`{"artifacts":{"size":0,"max-result":25,"start-index":0}}`)
	c := newTestClient(t, fb)
	page, err := c.ListPlanArtifacts(context.Background(), "PROJ-A", PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Results == nil || len(page.Results) != 0 {
		t.Fatalf("expected empty non-nil slice; got %+v", page.Results)
	}
}

func TestListPlanVCSBranches(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-A/vcsBranches", 200, `{
		"branches":{"size":3,"max-result":25,"start-index":0,
			"branch":[{"name":"main"},{"name":"dev"},{"name":"feature/x"}]
		}
	}`)
	c := newTestClient(t, fb)
	page, err := c.ListPlanVCSBranches(context.Background(), "PROJ-A", PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Results) != 3 || page.Results[0].Name != "main" {
		t.Errorf("results = %+v", page.Results)
	}
}

// TestListPlanVariablesEmptyTopLevelArray verifies the case where Bamboo
// returns an empty array (no variables defined).
func TestListPlanVariablesEmptyTopLevelArray(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/PROJ-PLAN/variables", 200, `[]`)
	c := newTestClient(t, fb)
	vs, err := c.ListPlanVariables(context.Background(), "PROJ-PLAN")
	if err != nil {
		t.Fatal(err)
	}
	if vs == nil || len(vs) != 0 {
		t.Fatalf("want empty non-nil slice, got %+v", vs)
	}
}
