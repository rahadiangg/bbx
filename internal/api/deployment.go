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
