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
