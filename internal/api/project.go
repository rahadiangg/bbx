package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Project is the parent of plans in Bamboo's domain model. A project owns one
// or more plans; project-scoped variables and repositories are inherited by
// every plan inside it.
type Project struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Link        Link   `json:"link,omitempty"`
}

// GetProject returns metadata for a single project.
func (c *Client) GetProject(ctx context.Context, projectKey string) (*Project, error) {
	var p Project
	if err := c.Do(ctx, http.MethodGet, "/api/latest/project/"+url.PathEscape(projectKey), nil, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ProjectSpec is the bulk Bamboo Specs export for an entire project. Unlike
// PlanSpec (which wraps one plan's source), this returns the Java source for
// every plan in the project, each as its own item under `spec`.
//
// Wire shape: `{"projectKey":"...","spec":[{"projectKey","buildKey","code"}, ...]}`.
type ProjectSpec struct {
	ProjectKey string     `json:"projectKey"`
	Spec       []PlanSpec `json:"spec"`
}

// GetProjectSpec fetches the bulk Specs export for a project — the Java
// source for every plan it contains. Useful when migrating an entire project
// rather than one plan at a time.
func (c *Client) GetProjectSpec(ctx context.Context, projectKey string) (*ProjectSpec, error) {
	var ps ProjectSpec
	if err := c.Do(ctx, http.MethodGet, "/api/latest/project/"+url.PathEscape(projectKey)+"/specs", nil, nil, &ps); err != nil {
		return nil, err
	}
	if ps.Spec == nil {
		ps.Spec = []PlanSpec{}
	}
	return &ps, nil
}

// ProjectVariable is a project-scoped variable. Same shape as PlanVariable —
// sensitive values are masked server-side as `********`.
type ProjectVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ListProjectVariables returns the project-scoped variables.
//
// Bamboo returns this as a top-level JSON array (NOT an envelope) — same
// pattern as ListPlanVariables. Pre-allocate `[]` for empty.
//
// NOTE for future writers: the POST to `/project/{key}/variable` (singular)
// to create a variable uses **query params**, not a JSON body — exactly like
// the plan variable endpoint. Not in this iteration's scope (list-only).
func (c *Client) ListProjectVariables(ctx context.Context, projectKey string) ([]ProjectVariable, error) {
	var out []ProjectVariable
	p := fmt.Sprintf("/api/latest/project/%s/variables", url.PathEscape(projectKey))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []ProjectVariable{}
	}
	return out, nil
}

// GetProjectVariable fetches one project variable by name.
func (c *Client) GetProjectVariable(ctx context.Context, projectKey, name string) (*ProjectVariable, error) {
	var v ProjectVariable
	p := fmt.Sprintf("/api/latest/project/%s/variable/%s", url.PathEscape(projectKey), url.PathEscape(name))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ProjectRepository describes a repository linked to a project (inherited by
// every plan inside that project).
type ProjectRepository struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"rssEnabled,omitempty"`
	Link        Link   `json:"link,omitempty"`
}

// ListProjectRepositories returns the repos available to plans in this project.
// Bamboo returns this as a top-level JSON array; pre-allocate `[]` for empty.
func (c *Client) ListProjectRepositories(ctx context.Context, projectKey string) ([]ProjectRepository, error) {
	var out []ProjectRepository
	p := fmt.Sprintf("/api/latest/project/%s/repository", url.PathEscape(projectKey))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []ProjectRepository{}
	}
	return out, nil
}
