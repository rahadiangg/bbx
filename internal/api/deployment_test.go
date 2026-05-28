package api

import (
	"context"
	"strings"
	"testing"
)

func TestListDeploymentQueuePopulated(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/queue/deployment", 200, `{
		"queuedDeployments":{"size":1,"start-index":0,"max-result":1,
			"queuedDeployment":[{"deploymentResultId":1,"environmentId":10,"environmentName":"prod"}]
		},
		"inProgress":{"size":1,"start-index":0,"max-result":1,
			"queuedDeployment":[{"deploymentResultId":2,"environmentId":11}]
		}
	}`)
	c := newTestClient(t, fb)
	q, err := c.ListDeploymentQueue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Queued) != 1 || q.Queued[0].EnvironmentName != "prod" {
		t.Errorf("queued = %+v", q.Queued)
	}
	if len(q.InProgress) != 1 || q.InProgress[0].EnvironmentID != 11 {
		t.Errorf("inProgress = %+v", q.InProgress)
	}
}

// TestListDeploymentQueueEmpty verifies the wire envelope Bamboo actually
// returns when nothing is queued (object with size=0 and no inner array).
func TestListDeploymentQueueEmpty(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/queue/deployment", 200, `{
		"queuedDeployments":{"size":0,"start-index":0,"max-result":0},
		"inProgress":{"size":0,"start-index":0,"max-result":0}
	}`)
	c := newTestClient(t, fb)
	q, err := c.ListDeploymentQueue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if q.Queued == nil || len(q.Queued) != 0 {
		t.Errorf("Queued should be empty slice (not nil), got %+v", q.Queued)
	}
	if q.InProgress == nil || len(q.InProgress) != 0 {
		t.Errorf("InProgress should be empty slice (not nil), got %+v", q.InProgress)
	}
}

func TestTriggerDeployment(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("POST", "/rest/api/latest/queue/deployment", 200, `{"id":42,"deploymentState":"PENDING"}`)
	c := newTestClient(t, fb)
	dr, err := c.TriggerDeployment(context.Background(), 7, 11)
	if err != nil {
		t.Fatal(err)
	}
	if dr.ID != 42 {
		t.Errorf("id = %d", dr.ID)
	}
	if !strings.Contains(rec.RawQuery, "environmentId=7") || !strings.Contains(rec.RawQuery, "versionId=11") {
		t.Errorf("query = %q", rec.RawQuery)
	}
}

func TestCancelDeployment(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("DELETE", "/rest/api/latest/queue/deployment/42", 204, "")
	c := newTestClient(t, fb)
	if err := c.CancelDeployment(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if rec.Method != "DELETE" {
		t.Errorf("method = %q", rec.Method)
	}
}

func TestGetDeploymentResult(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/result/42", 200, `{"id":42,"deploymentState":"SUCCEEDED"}`)
	c := newTestClient(t, fb)
	dr, err := c.GetDeploymentResult(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if dr.DeploymentState != "SUCCEEDED" {
		t.Errorf("state = %q", dr.DeploymentState)
	}
}

func TestPreviewDeploymentVersion(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("GET", "/rest/api/latest/deploy/preview/version", 200, `{"nextVersionName":"v2","previousVersionName":"v1"}`)
	c := newTestClient(t, fb)
	vp, err := c.PreviewDeploymentVersion(context.Background(), "5", "PROJ-PLAN-1")
	if err != nil {
		t.Fatal(err)
	}
	if vp.NextVersionName != "v2" {
		t.Errorf("next = %q", vp.NextVersionName)
	}
	if !strings.Contains(rec.RawQuery, "deploymentProjectId=5") {
		t.Errorf("query = %q", rec.RawQuery)
	}
	if !strings.Contains(rec.RawQuery, "planResultKey=PROJ-PLAN-1") {
		t.Errorf("query = %q", rec.RawQuery)
	}
}

func TestPreviewDeploymentVersionEmptyQuery(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("GET", "/rest/api/latest/deploy/preview/version", 200, `{"nextVersionName":"v2"}`)
	c := newTestClient(t, fb)
	if _, err := c.PreviewDeploymentVersion(context.Background(), "", ""); err != nil {
		t.Fatal(err)
	}
	if rec.RawQuery != "" {
		t.Errorf("expected empty query, got %q", rec.RawQuery)
	}
}

// Deployment config-extraction tests -----------------------------------------

func TestListDeploymentProjects(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/all", 200,
		`[{"id":1,"name":"DP1","planKey":{"key":"PROJ-A"}},{"id":2,"name":"DP2","planKey":{"key":"PROJ-B"}}]`)
	c := newTestClient(t, fb)
	dps, err := c.ListDeploymentProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(dps) != 2 || dps[0].Name != "DP1" || dps[1].ID != 2 {
		t.Errorf("dps = %+v", dps)
	}
}

func TestListDeploymentProjectsForPlan(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("GET", "/rest/api/latest/deploy/project/forPlan", 200,
		`[{"id":7,"name":"DPlinked"}]`)
	c := newTestClient(t, fb)
	dps, err := c.ListDeploymentProjectsForPlan(context.Background(), "PROJ-A")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rec.RawQuery, "planKey=PROJ-A") {
		t.Errorf("query missing planKey: %q", rec.RawQuery)
	}
	if len(dps) != 1 || dps[0].ID != 7 {
		t.Errorf("dps = %+v", dps)
	}
}

func TestListDeploymentProjectsForPlanEmpty(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/forPlan", 200, `[]`)
	c := newTestClient(t, fb)
	dps, err := c.ListDeploymentProjectsForPlan(context.Background(), "PROJ-A")
	if err != nil {
		t.Fatal(err)
	}
	if dps == nil || len(dps) != 0 {
		t.Fatalf("expected non-nil empty slice; got %+v", dps)
	}
}

func TestGetDeploymentProject(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/42", 200,
		`{"id":42,"name":"Prod","environments":[{"id":100,"name":"prod-env"}]}`)
	c := newTestClient(t, fb)
	dp, err := c.GetDeploymentProject(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if dp.ID != 42 || dp.Name != "Prod" || len(dp.Environments) != 1 {
		t.Fatalf("dp = %+v", dp)
	}
}

func TestGetDeploymentProjectSpec(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/42/specs", 200,
		`{"deploymentId":42,"code":"import com.atlassian.bamboo.specs..."}`)
	c := newTestClient(t, fb)
	sp, err := c.GetDeploymentProjectSpec(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if sp.DeploymentID != 42 || !strings.Contains(sp.Code, "bamboo.specs") {
		t.Errorf("sp = %+v", sp)
	}
}

func TestListDeploymentProjectRepositories(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/42/repository", 200,
		`[{"id":1,"name":"dr1"}]`)
	c := newTestClient(t, fb)
	rs, err := c.ListDeploymentProjectRepositories(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 || rs[0].Name != "dr1" {
		t.Errorf("rs = %+v", rs)
	}
}

func TestGetDeploymentEnvironment(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/environment/100", 200,
		`{"id":100,"name":"prod-env","configurationState":"VALID"}`)
	c := newTestClient(t, fb)
	e, err := c.GetDeploymentEnvironment(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if e.ID != 100 || e.Name != "prod-env" || e.ConfigurationState != "VALID" {
		t.Errorf("env = %+v", e)
	}
}

func TestListEnvironmentVariables(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/environment/100/variables", 200,
		`[{"id":1,"key":"K","value":"v"}]`)
	c := newTestClient(t, fb)
	vs, err := c.ListEnvironmentVariables(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].Key != "K" {
		t.Errorf("vs = %+v", vs)
	}
}

func TestListEnvironmentRequirements(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/environment/100/requirement", 200,
		`[{"id":1,"key":"os","matchType":"EQUALS","matchValue":"linux"}]`)
	c := newTestClient(t, fb)
	rs, err := c.ListEnvironmentRequirements(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 || rs[0].MatchValue != "linux" {
		t.Errorf("rs = %+v", rs)
	}
}

func TestListEnvironmentAgentAssignments(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/environment/100/agent-assignment", 200,
		`[{"executorId":5,"executorType":"AGENT","executableId":100,"executableType":"ENVIRONMENT"}]`)
	c := newTestClient(t, fb)
	as, err := c.ListEnvironmentAgentAssignments(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 1 || as[0].ExecutorID != 5 {
		t.Errorf("as = %+v", as)
	}
}

func TestListDeploymentVersions(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/deploy/project/42/versions", 200,
		`{"size":2,"versions":[{"id":1,"name":"v1","creatorUserName":"a"},{"id":2,"name":"v2"}]}`)
	c := newTestClient(t, fb)
	page, err := c.ListDeploymentVersions(context.Background(), 42, PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Results) != 2 || page.Results[0].Name != "v1" {
		t.Errorf("results = %+v", page.Results)
	}
}

func TestWhoAmI(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/currentUser", 200, `{"name":"jdoe","fullName":"Jane Doe","email":"j@d.com"}`)
	c := newTestClient(t, fb)
	u, err := c.WhoAmI(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if u.Name != "jdoe" || u.FullName != "Jane Doe" || u.Email != "j@d.com" {
		t.Errorf("user = %+v", u)
	}
}
