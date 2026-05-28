package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// DeploymentQueueItem represents a deployment in the queue.
type DeploymentQueueItem struct {
	DeploymentResultID  int64  `json:"deploymentResultId"`
	EnvironmentID       int64  `json:"environmentId"`
	EnvironmentName     string `json:"environmentName,omitempty"`
	DeploymentVersionID int64  `json:"deploymentVersionId,omitempty"`
	UserName            string `json:"userName,omitempty"`
}

// DeploymentQueue is the user-facing shape of the deployment queue.
type DeploymentQueue struct {
	Queued     []DeploymentQueueItem `json:"queued"`
	InProgress []DeploymentQueueItem `json:"inProgress"`
}

// rawDeploymentQueueEnvelope mirrors Bamboo's actual wire shape: each section
// is an envelope object with `size`, `start-index`, `max-result`, and the
// items under a singular key (e.g. `queuedDeployment`) when non-empty.
//
// IMPORTANT: even when empty, Bamboo returns `{queuedDeployments: {size: 0}}`
// — NOT a bare array. Decoding into `[]DeploymentQueueItem` (as bbx did
// briefly) produces a JSON type-mismatch error against real servers.
// Regression-tested in TestListDeploymentQueueEmpty.
type rawDeploymentQueueEnvelope struct {
	QueuedDeployments struct {
		Size             int                   `json:"size"`
		QueuedDeployment []DeploymentQueueItem `json:"queuedDeployment"`
	} `json:"queuedDeployments"`
	InProgress struct {
		Size             int                   `json:"size"`
		QueuedDeployment []DeploymentQueueItem `json:"queuedDeployment"`
	} `json:"inProgress"`
}

// ListDeploymentQueue returns the deployment queue (queued + in-progress).
// Both slices are always non-nil so JSON output is `[]` rather than `null`.
func (c *Client) ListDeploymentQueue(ctx context.Context) (*DeploymentQueue, error) {
	var raw rawDeploymentQueueEnvelope
	if err := c.Do(ctx, http.MethodGet, "/api/latest/queue/deployment", nil, nil, &raw); err != nil {
		return nil, err
	}
	out := &DeploymentQueue{
		Queued:     raw.QueuedDeployments.QueuedDeployment,
		InProgress: raw.InProgress.QueuedDeployment,
	}
	if out.Queued == nil {
		out.Queued = []DeploymentQueueItem{}
	}
	if out.InProgress == nil {
		out.InProgress = []DeploymentQueueItem{}
	}
	return out, nil
}

// DeploymentResult is the outcome of a single deployment.
type DeploymentResult struct {
	ID                int64  `json:"id,omitempty"`
	DeploymentVersion any    `json:"deploymentVersion,omitempty"`
	DeploymentState   string `json:"deploymentState,omitempty"`
	LifeCycleState    string `json:"lifeCycleState,omitempty"`
	StartedDate       string `json:"startedDate,omitempty"`
	FinishedDate      string `json:"finishedDate,omitempty"`
	Reason            string `json:"reason,omitempty"`
	Environment       any    `json:"environment,omitempty"`
}

// TriggerDeployment enqueues a deployment of `versionID` to `environmentID`.
func (c *Client) TriggerDeployment(ctx context.Context, environmentID, versionID int64) (*DeploymentResult, error) {
	q := url.Values{}
	q.Set("environmentId", strconv.FormatInt(environmentID, 10))
	q.Set("versionId", strconv.FormatInt(versionID, 10))
	var dr DeploymentResult
	if err := c.Do(ctx, http.MethodPost, "/api/latest/queue/deployment", q, nil, &dr); err != nil {
		return nil, err
	}
	return &dr, nil
}

// CancelDeployment removes a queued deployment.
func (c *Client) CancelDeployment(ctx context.Context, deploymentResultID int64) error {
	p := fmt.Sprintf("/api/latest/queue/deployment/%d", deploymentResultID)
	return c.Do(ctx, http.MethodDelete, p, nil, nil, nil)
}

// GetDeploymentResult fetches a single deployment by ID.
func (c *Client) GetDeploymentResult(ctx context.Context, id int64) (*DeploymentResult, error) {
	var dr DeploymentResult
	p := fmt.Sprintf("/api/latest/deploy/result/%d", id)
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &dr); err != nil {
		return nil, err
	}
	return &dr, nil
}

// VersionPreview previews what version would be deployed.
type VersionPreview struct {
	NextVersionName     string `json:"nextVersionName,omitempty"`
	PreviousVersionName string `json:"previousVersionName,omitempty"`
}

func (c *Client) PreviewDeploymentVersion(ctx context.Context, deploymentProjectID, planResultKey string) (*VersionPreview, error) {
	q := url.Values{}
	if deploymentProjectID != "" {
		q.Set("deploymentProjectId", deploymentProjectID)
	}
	if planResultKey != "" {
		q.Set("planResultKey", planResultKey)
	}
	var vp VersionPreview
	if err := c.Do(ctx, http.MethodGet, "/api/latest/deploy/preview/version", q, nil, &vp); err != nil {
		return nil, err
	}
	return &vp, nil
}

// Deployment configuration extraction --------------------------------------

// DeploymentProject is the parent of deployment environments + versions.
type DeploymentProject struct {
	ID           int64                   `json:"id"`
	OID          string                  `json:"oid,omitempty"`
	Name         string                  `json:"name"`
	Description  string                  `json:"description,omitempty"`
	PlanKey      map[string]any          `json:"planKey,omitempty"` // {"key":"PROJ-PLAN"}
	Key          map[string]any          `json:"key,omitempty"`     // {"key":"12345"}
	Environments []DeploymentEnvironment `json:"environments,omitempty"`
}

// ListDeploymentProjects returns every deployment project visible to the
// caller. Bamboo serves this as a top-level JSON array (not the standard
// paginated envelope); pre-allocate `[]` for empty.
func (c *Client) ListDeploymentProjects(ctx context.Context) ([]DeploymentProject, error) {
	var out []DeploymentProject
	if err := c.Do(ctx, http.MethodGet, "/api/latest/deploy/project/all", nil, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []DeploymentProject{}
	}
	return out, nil
}

// ListDeploymentProjectsForPlan returns the deployment projects linked to a
// build plan (one plan can feed multiple deployment projects, though one is
// typical). Returns `[]` (not an error) when nothing is linked.
func (c *Client) ListDeploymentProjectsForPlan(ctx context.Context, planKey string) ([]DeploymentProject, error) {
	q := url.Values{}
	q.Set("planKey", planKey)
	var out []DeploymentProject
	if err := c.Do(ctx, http.MethodGet, "/api/latest/deploy/project/forPlan", q, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []DeploymentProject{}
	}
	return out, nil
}

// GetDeploymentProject returns full configuration for one deployment project
// including its embedded environments slice.
func (c *Client) GetDeploymentProject(ctx context.Context, id int64) (*DeploymentProject, error) {
	var dp DeploymentProject
	p := fmt.Sprintf("/api/latest/deploy/project/%d", id)
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &dp); err != nil {
		return nil, err
	}
	return &dp, nil
}

// DeploymentProjectSpec wraps the Bamboo Specs Java source for a deployment
// project. Wire shape differs from PlanSpec: `{"deploymentId":N,"code":"..."}`
// (no nested `spec` object).
type DeploymentProjectSpec struct {
	DeploymentID int64  `json:"deploymentId"`
	Code         string `json:"code"`
}

// GetDeploymentProjectSpec returns the Bamboo Specs Java source for a
// deployment project — the full environment/release config as executable
// Java. Complement to GetPlanSpec for the deploy side.
func (c *Client) GetDeploymentProjectSpec(ctx context.Context, id int64) (*DeploymentProjectSpec, error) {
	var sp DeploymentProjectSpec
	p := fmt.Sprintf("/api/latest/deploy/project/%d/specs", id)
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &sp); err != nil {
		return nil, err
	}
	return &sp, nil
}

// DeploymentProjectRepository is a repo linked to a deployment project.
type DeploymentProjectRepository struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name"`
	Link Link   `json:"link,omitempty"`
}

// ListDeploymentProjectRepositories returns repos linked to a deployment
// project. Top-level JSON array; pre-allocate `[]` for empty.
func (c *Client) ListDeploymentProjectRepositories(ctx context.Context, id int64) ([]DeploymentProjectRepository, error) {
	var out []DeploymentProjectRepository
	p := fmt.Sprintf("/api/latest/deploy/project/%d/repository", id)
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []DeploymentProjectRepository{}
	}
	return out, nil
}

// DeploymentEnvironment is one target environment within a deployment project
// (e.g. "Dev", "Staging", "Prod"). Bamboo embeds these in DeploymentProject.
type DeploymentEnvironment struct {
	ID                 int64          `json:"id"`
	Key                map[string]any `json:"key,omitempty"`
	Name               string         `json:"name"`
	Description        string         `json:"description,omitempty"`
	Position           int            `json:"position,omitempty"`
	ConfigurationState string         `json:"configurationState,omitempty"`
	OperationsCount    int            `json:"operationsCount,omitempty"`
}

// GetDeploymentEnvironment returns one environment by ID.
func (c *Client) GetDeploymentEnvironment(ctx context.Context, envID int64) (*DeploymentEnvironment, error) {
	var e DeploymentEnvironment
	p := fmt.Sprintf("/api/latest/deploy/environment/%d", envID)
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// EnvVariable is an environment-scoped deployment variable.
type EnvVariable struct {
	ID    int64  `json:"id,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// ListEnvironmentVariables returns the variables defined on a deployment env.
// Bamboo serves these as a top-level JSON array.
func (c *Client) ListEnvironmentVariables(ctx context.Context, envID int64) ([]EnvVariable, error) {
	var out []EnvVariable
	p := fmt.Sprintf("/api/latest/deploy/environment/%d/variables", envID)
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []EnvVariable{}
	}
	return out, nil
}

// EnvRequirement is one agent-capability requirement.
type EnvRequirement struct {
	ID         int64  `json:"id,omitempty"`
	Key        string `json:"key,omitempty"`
	MatchType  string `json:"matchType,omitempty"`
	MatchValue string `json:"matchValue,omitempty"`
}

// ListEnvironmentRequirements returns the agent-capability requirements for
// an environment (the "what kind of agent must run this" rules).
func (c *Client) ListEnvironmentRequirements(ctx context.Context, envID int64) ([]EnvRequirement, error) {
	var out []EnvRequirement
	p := fmt.Sprintf("/api/latest/deploy/environment/%d/requirement", envID)
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []EnvRequirement{}
	}
	return out, nil
}

// AgentAssignment is one agent or Docker image assigned to an environment.
type AgentAssignment struct {
	ExecutorID     int64  `json:"executorId,omitempty"`
	ExecutorType   string `json:"executorType,omitempty"` // "AGENT" or "IMAGE"
	ExecutableID   int64  `json:"executableId,omitempty"`
	ExecutableType string `json:"executableType,omitempty"`
}

// ListEnvironmentAgentAssignments returns the agents (or Docker images)
// assigned to a deployment environment.
func (c *Client) ListEnvironmentAgentAssignments(ctx context.Context, envID int64) ([]AgentAssignment, error) {
	var out []AgentAssignment
	p := fmt.Sprintf("/api/latest/deploy/environment/%d/agent-assignment", envID)
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []AgentAssignment{}
	}
	return out, nil
}

// DeploymentVersion is one immutable snapshot of a deployment project's
// inputs (typically a build's artifacts) ready to be deployed to an env.
type DeploymentVersion struct {
	ID              int64            `json:"id"`
	Name            string           `json:"name"`
	CreationDate    int64            `json:"creationDate,omitempty"` // epoch ms
	CreatorUserName string           `json:"creatorUserName,omitempty"`
	Items           []map[string]any `json:"items,omitempty"`
}

// deploymentVersionsEnvelope wraps Bamboo's bespoke `{size,versions:[...]}`
// shape — distinct from both the standard paginated envelope and the bare
// array used by repos/variables.
type deploymentVersionsEnvelope struct {
	Size     int                 `json:"size"`
	Versions []DeploymentVersion `json:"versions"`
}

// ListDeploymentVersions returns versions for a deployment project. Pagination
// uses the standard PageOpts (start-index + max-results) but the response
// shape is unique to this endpoint.
func (c *Client) ListDeploymentVersions(ctx context.Context, projectID int64, opts PageOpts) (Page[DeploymentVersion], error) {
	var env deploymentVersionsEnvelope
	p := fmt.Sprintf("/api/latest/deploy/project/%d/versions", projectID)
	if err := c.Do(ctx, http.MethodGet, p, opts.Values(), nil, &env); err != nil {
		return Page[DeploymentVersion]{}, err
	}
	results := env.Versions
	if results == nil {
		results = []DeploymentVersion{}
	}
	return Page[DeploymentVersion]{
		Results:    results,
		Size:       env.Size,
		MaxResult:  opts.MaxResults,
		StartIndex: opts.StartIndex,
	}, nil
}

// CurrentUser maps GET /api/latest/currentUser used for auth whoami.
type CurrentUser struct {
	Name        string `json:"name"`
	Email       string `json:"email,omitempty"`
	FullName    string `json:"fullName,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

func (c *Client) WhoAmI(ctx context.Context) (*CurrentUser, error) {
	var u CurrentUser
	if err := c.Do(ctx, http.MethodGet, "/api/latest/currentUser", nil, nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}
