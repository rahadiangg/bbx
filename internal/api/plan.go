package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Plan represents a Bamboo build plan (pipeline).
type Plan struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName,omitempty"`
	ShortKey    string `json:"shortKey,omitempty"`
	Type        string `json:"type,omitempty"`
	Enabled     bool   `json:"enabled"`
	ProjectKey  string `json:"projectKey,omitempty"`
	ProjectName string `json:"projectName,omitempty"`
	Description string `json:"description,omitempty"`
	Link        Link   `json:"link,omitempty"`
}

type Link struct {
	Href string `json:"href,omitempty"`
	Rel  string `json:"rel,omitempty"`
}

// planEnvelope is the wire shape of GET /api/latest/plan.
type planEnvelope struct {
	Expand string `json:"expand"`
	Plans  struct {
		Size       int    `json:"size"`
		MaxResult  int    `json:"max-result"`
		StartIndex int    `json:"start-index"`
		Plan       []Plan `json:"plan"`
	} `json:"plans"`
}

// ListPlans returns one page of plans.
func (c *Client) ListPlans(ctx context.Context, opts PageOpts) (Page[Plan], error) {
	if opts.Expand == "" {
		opts.Expand = "plans"
	}
	var env planEnvelope
	if err := c.Do(ctx, http.MethodGet, "/api/latest/plan", opts.Values(), nil, &env); err != nil {
		return Page[Plan]{}, err
	}
	return Page[Plan]{
		Results:    env.Plans.Plan,
		Size:       env.Plans.Size,
		MaxResult:  env.Plans.MaxResult,
		StartIndex: env.Plans.StartIndex,
	}, nil
}

// GetPlan returns full details for a single plan.
func (c *Client) GetPlan(ctx context.Context, key string) (*Plan, error) {
	var p Plan
	if err := c.Do(ctx, http.MethodGet, "/api/latest/plan/"+url.PathEscape(key), nil, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) EnablePlan(ctx context.Context, key string) error {
	return c.Do(ctx, http.MethodPost, "/api/latest/plan/"+url.PathEscape(key)+"/enable", nil, nil, nil)
}

func (c *Client) DisablePlan(ctx context.Context, key string) error {
	return c.Do(ctx, http.MethodDelete, "/api/latest/plan/"+url.PathEscape(key)+"/enable", nil, nil, nil)
}

func (c *Client) DeletePlan(ctx context.Context, key string) error {
	return c.Do(ctx, http.MethodDelete, "/api/latest/plan/"+url.PathEscape(key), nil, nil, nil)
}

// ClonePlan creates a new plan by cloning an existing one (Bamboo's only REST
// path to "create" a plan — there is no POST /plan endpoint as of 8.2.4).
//
// Preconditions:
//   - the destination project must already exist
//   - the destination plan key must NOT exist (Bamboo returns 4xx otherwise)
//
// Path note: Bamboo expects the src and dst keys joined by a *literal* colon,
// e.g. `/clone/RES-TES:RES-BBX`. url.PathEscape is applied to src/dst
// individually but the joining colon stays unencoded — encoding it as %3A
// causes a 404 because Bamboo's router only matches the literal form.
//
// Response shape varies across Bamboo versions: 8.2.4 returns the new Plan
// object, but other versions have been observed returning 204 No Content or
// a slim envelope. We try a strict decode into Plan first; on decode failure
// we fall back to `map[string]any` so the caller still sees whatever the
// server returned rather than getting a hard error on version drift.
func (c *Client) ClonePlan(ctx context.Context, src, dst string) (any, error) {
	if src == "" || dst == "" {
		return nil, fmt.Errorf("clone: src and dst keys are required")
	}
	path := "/api/latest/clone/" + url.PathEscape(src) + ":" + url.PathEscape(dst)
	body, err := c.doRawJSON(ctx, http.MethodPut, path, nil, nil)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		// Some Bamboo versions return 204 No Content; surface a minimal record.
		return map[string]any{"key": dst}, nil
	}
	var p Plan
	if err := json.Unmarshal(body, &p); err == nil && p.Key != "" {
		return &p, nil
	}
	// Fallback: caller still sees the server's response in JSON output mode.
	var generic map[string]any
	if err := json.Unmarshal(body, &generic); err == nil {
		return generic, nil
	}
	// Last resort: return the raw payload as a string so nothing is lost.
	return map[string]any{"key": dst, "raw_response": string(body)}, nil
}

// DeletePlanBranch removes a plan branch by name.
//
// IMPORTANT: Bamboo Server 8.2.4 does NOT expose DELETE on
// `/plan/{key}/branch/{name}` (verified via OPTIONS — that path allows only
// GET and PUT). Instead, each plan branch has its own auto-generated plan
// key (e.g. branch "feature/x" of plan PROJ-A becomes plan PROJ-A0). To
// delete the branch, we look the branch up to discover its plan key, then
// call DELETE on that key — the same endpoint that deletes regular plans.
func (c *Client) DeletePlanBranch(ctx context.Context, planKey, branchName string) error {
	br, err := c.GetPlanBranch(ctx, planKey, branchName)
	if err != nil {
		return err
	}
	if br.Key == "" {
		return fmt.Errorf("plan branch %q on %s: server returned no key for the branch", branchName, planKey)
	}
	return c.DeletePlan(ctx, br.Key)
}

// Plan branches ----------------------------------------------------------

type PlanBranch struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName,omitempty"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
	Link        Link   `json:"link,omitempty"`
}

type branchesEnvelope struct {
	Branches struct {
		Size       int          `json:"size"`
		MaxResult  int          `json:"max-result"`
		StartIndex int          `json:"start-index"`
		Branch     []PlanBranch `json:"branch"`
	} `json:"branches"`
}

func (c *Client) ListPlanBranches(ctx context.Context, planKey string, opts PageOpts) (Page[PlanBranch], error) {
	if opts.Expand == "" {
		opts.Expand = "branches"
	}
	var env branchesEnvelope
	p := fmt.Sprintf("/api/latest/plan/%s/branch", url.PathEscape(planKey))
	if err := c.Do(ctx, http.MethodGet, p, opts.Values(), nil, &env); err != nil {
		return Page[PlanBranch]{}, err
	}
	return Page[PlanBranch]{
		Results:    env.Branches.Branch,
		Size:       env.Branches.Size,
		MaxResult:  env.Branches.MaxResult,
		StartIndex: env.Branches.StartIndex,
	}, nil
}

func (c *Client) GetPlanBranch(ctx context.Context, planKey, branchName string) (*PlanBranch, error) {
	var pb PlanBranch
	p := fmt.Sprintf("/api/latest/plan/%s/branch/%s", url.PathEscape(planKey), url.PathEscape(branchName))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

// CreatePlanBranch creates a new plan branch tracking the given VCS branch.
func (c *Client) CreatePlanBranch(ctx context.Context, planKey, branchName, vcsBranch string) (*PlanBranch, error) {
	q := url.Values{}
	if vcsBranch != "" {
		q.Set("vcsBranch", vcsBranch)
	}
	var pb PlanBranch
	p := fmt.Sprintf("/api/latest/plan/%s/branch/%s", url.PathEscape(planKey), url.PathEscape(branchName))
	if err := c.Do(ctx, http.MethodPut, p, q, nil, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

// Plan variables ---------------------------------------------------------

// PlanVariable is a single plan-scoped variable. Bamboo Server returns these
// as a top-level JSON array using `name`/`value` keys; sensitive values are
// masked server-side as `********` rather than exposed via a flag.
type PlanVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (c *Client) ListPlanVariables(ctx context.Context, planKey string) ([]PlanVariable, error) {
	var out []PlanVariable
	p := fmt.Sprintf("/api/latest/plan/%s/variables", url.PathEscape(planKey))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []PlanVariable{}
	}
	return out, nil
}

func (c *Client) GetPlanVariable(ctx context.Context, planKey, name string) (*PlanVariable, error) {
	var v PlanVariable
	p := fmt.Sprintf("/api/latest/plan/%s/variables/%s", url.PathEscape(planKey), url.PathEscape(name))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// AddPlanVariable creates a new variable. Use UpdatePlanVariable for existing ones.
//
// IMPORTANT: Bamboo 8.2.4 expects `name` and `value` as URL **query parameters**,
// not a JSON body. Sending JSON returns 400 "You must enter a valid variable
// name". The corresponding PUT (update) does take a JSON body — the two
// endpoints are intentionally asymmetric. Discovered by direct probing.
func (c *Client) AddPlanVariable(ctx context.Context, planKey, name, value string) (*PlanVariable, error) {
	q := url.Values{}
	q.Set("name", name)
	q.Set("value", value)
	var v PlanVariable
	p := fmt.Sprintf("/api/latest/plan/%s/variables", url.PathEscape(planKey))
	if err := c.Do(ctx, http.MethodPost, p, q, nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) UpdatePlanVariable(ctx context.Context, planKey, name, value string) (*PlanVariable, error) {
	body := PlanVariable{Name: name, Value: value}
	var v PlanVariable
	p := fmt.Sprintf("/api/latest/plan/%s/variables/%s", url.PathEscape(planKey), url.PathEscape(name))
	if err := c.Do(ctx, http.MethodPut, p, nil, body, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) DeletePlanVariable(ctx context.Context, planKey, name string) error {
	p := fmt.Sprintf("/api/latest/plan/%s/variables/%s", url.PathEscape(planKey), url.PathEscape(name))
	return c.Do(ctx, http.MethodDelete, p, nil, nil, nil)
}

// Plan configuration extraction --------------------------------------------

// PlanSpec wraps the Bamboo Specs Java source for a single plan. Bamboo
// auto-generates the source from the live plan configuration — it does NOT
// require the plan to have been originally created via Bamboo Specs.
type PlanSpec struct {
	ProjectKey string `json:"projectKey"`
	BuildKey   string `json:"buildKey"`
	Code       string `json:"code"` // executable Bamboo Specs Java
}

type planSpecEnvelope struct {
	Spec PlanSpec `json:"spec"`
}

// GetPlanSpec fetches the full Bamboo Specs Java source for a plan. This is
// the most complete representation of a plan available via REST: stages,
// jobs, tasks (executable + arguments), variables, branch management,
// permissions — everything an AI agent needs to replicate the plan in
// another CI system.
func (c *Client) GetPlanSpec(ctx context.Context, planKey string) (*PlanSpec, error) {
	var env planSpecEnvelope
	p := fmt.Sprintf("/api/latest/plan/%s/specs", url.PathEscape(planKey))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &env); err != nil {
		return nil, err
	}
	return &env.Spec, nil
}

// GetPlanConfig fetches the plan with maximum expand — stages, jobs, actions,
// branches, variableContext — as a structured JSON document. Complementary to
// GetPlanSpec: the spec endpoint returns Java source, this one returns the
// same information as nested JSON, which is easier to programmatically inspect
// without a Java parser.
//
// We return map[string]any rather than a strict struct because the expanded
// shape is large, nested, and varies subtly across Bamboo versions. The caller
// (or its AI agent) inspects fields directly.
func (c *Client) GetPlanConfig(ctx context.Context, planKey string) (map[string]any, error) {
	q := url.Values{}
	q.Set("expand", "stages.stage.plans.plan,actions,variableContext,branches")
	var out map[string]any
	p := "/api/latest/plan/" + url.PathEscape(planKey)
	if err := c.Do(ctx, http.MethodGet, p, q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PlanArtifact describes one artifact definition on a plan (a file pattern
// the plan produces and optionally shares between jobs).
type PlanArtifact struct {
	Name           string `json:"name"`
	Location       string `json:"location,omitempty"`
	CopyPattern    string `json:"copyPattern,omitempty"`
	Shared         bool   `json:"shared,omitempty"`
	Required       bool   `json:"required,omitempty"`
	ProducerJobKey any    `json:"producerJobKey,omitempty"`
	Link           Link   `json:"link,omitempty"`
}

type planArtifactsEnvelope struct {
	Artifacts struct {
		Size       int            `json:"size"`
		MaxResult  int            `json:"max-result"`
		StartIndex int            `json:"start-index"`
		Artifact   []PlanArtifact `json:"artifact"`
	} `json:"artifacts"`
}

// ListPlanArtifacts returns the artifact definitions for a plan.
func (c *Client) ListPlanArtifacts(ctx context.Context, planKey string, opts PageOpts) (Page[PlanArtifact], error) {
	var env planArtifactsEnvelope
	p := fmt.Sprintf("/api/latest/plan/%s/artifact", url.PathEscape(planKey))
	if err := c.Do(ctx, http.MethodGet, p, opts.Values(), nil, &env); err != nil {
		return Page[PlanArtifact]{}, err
	}
	results := env.Artifacts.Artifact
	if results == nil {
		results = []PlanArtifact{}
	}
	return Page[PlanArtifact]{
		Results:    results,
		Size:       env.Artifacts.Size,
		MaxResult:  env.Artifacts.MaxResult,
		StartIndex: env.Artifacts.StartIndex,
	}, nil
}

// VCSBranch is a branch that exists in the plan's underlying VCS repository
// (NOT a Bamboo "plan branch" — that's a separate concept exposed by
// ListPlanBranches).
type VCSBranch struct {
	Name string `json:"name"`
}

type planVCSBranchesEnvelope struct {
	Branches struct {
		Size       int         `json:"size"`
		MaxResult  int         `json:"max-result"`
		StartIndex int         `json:"start-index"`
		Branch     []VCSBranch `json:"branch"`
	} `json:"branches"`
}

// ListPlanVCSBranches returns the branches present in the underlying repository
// for a plan. Useful for syncing plan branches with the upstream repo state.
func (c *Client) ListPlanVCSBranches(ctx context.Context, planKey string, opts PageOpts) (Page[VCSBranch], error) {
	var env planVCSBranchesEnvelope
	p := fmt.Sprintf("/api/latest/plan/%s/vcsBranches", url.PathEscape(planKey))
	if err := c.Do(ctx, http.MethodGet, p, opts.Values(), nil, &env); err != nil {
		return Page[VCSBranch]{}, err
	}
	results := env.Branches.Branch
	if results == nil {
		results = []VCSBranch{}
	}
	return Page[VCSBranch]{
		Results:    results,
		Size:       env.Branches.Size,
		MaxResult:  env.Branches.MaxResult,
		StartIndex: env.Branches.StartIndex,
	}, nil
}
