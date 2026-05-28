package api

import (
	"context"
	"strings"
	"testing"
)

func TestGetProject(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/ANVIS", 200,
		`{"key":"ANVIS","name":"Anvis","description":""}`)
	c := newTestClient(t, fb)
	p, err := c.GetProject(context.Background(), "ANVIS")
	if err != nil {
		t.Fatal(err)
	}
	if p.Key != "ANVIS" || p.Name != "Anvis" {
		t.Fatalf("project = %+v", p)
	}
}

func TestGetProjectSpec(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	// Bulk shape: {projectKey, spec:[{projectKey, buildKey, code}, ...]}.
	fb.expect("GET", "/rest/api/latest/project/PROJ/specs", 200,
		`{"projectKey":"PROJ","spec":[{"projectKey":"PROJ","buildKey":"A","code":"// plan A"},{"projectKey":"PROJ","buildKey":"B","code":"// plan B"}]}`)
	c := newTestClient(t, fb)
	ps, err := c.GetProjectSpec(context.Background(), "PROJ")
	if err != nil {
		t.Fatal(err)
	}
	if ps.ProjectKey != "PROJ" || len(ps.Spec) != 2 {
		t.Fatalf("spec = %+v", ps)
	}
	if !strings.Contains(ps.Spec[0].Code, "plan A") {
		t.Errorf("Spec[0].Code = %q", ps.Spec[0].Code)
	}
}

func TestGetProjectSpecEmpty(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/PROJ/specs", 200, `{"projectKey":"PROJ"}`)
	c := newTestClient(t, fb)
	ps, err := c.GetProjectSpec(context.Background(), "PROJ")
	if err != nil {
		t.Fatal(err)
	}
	if ps.Spec == nil || len(ps.Spec) != 0 {
		t.Fatalf("expected non-nil empty slice; got %+v", ps.Spec)
	}
}

func TestListProjectVariables(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/ANVIS/variables", 200,
		`[{"name":"PROJ_VAR","value":"v"},{"name":"SECRET","value":"********"}]`)
	c := newTestClient(t, fb)
	vs, err := c.ListProjectVariables(context.Background(), "ANVIS")
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 2 || vs[0].Name != "PROJ_VAR" || vs[1].Value != "********" {
		t.Errorf("vars = %+v", vs)
	}
}

func TestListProjectVariablesEmpty(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/ANVIS/variables", 200, `[]`)
	c := newTestClient(t, fb)
	vs, err := c.ListProjectVariables(context.Background(), "ANVIS")
	if err != nil {
		t.Fatal(err)
	}
	if vs == nil || len(vs) != 0 {
		t.Fatalf("expected non-nil empty slice; got %+v", vs)
	}
}

func TestGetProjectVariable(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/ANVIS/variable/MY_VAR", 200,
		`{"name":"MY_VAR","value":"hello"}`)
	c := newTestClient(t, fb)
	v, err := c.GetProjectVariable(context.Background(), "ANVIS", "MY_VAR")
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != "MY_VAR" || v.Value != "hello" {
		t.Errorf("v = %+v", v)
	}
}

func TestListProjectRepositories(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/project/ANVIS/repository", 200,
		`[{"id":42,"name":"core-repo","description":"main"}]`)
	c := newTestClient(t, fb)
	rs, err := c.ListProjectRepositories(context.Background(), "ANVIS")
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 || rs[0].ID != 42 || rs[0].Name != "core-repo" {
		t.Errorf("repos = %+v", rs)
	}
}
