package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rahadiangg/bbx/internal/fail"
)

// QueuedBuild describes an entry in /api/latest/queue.
type QueuedBuild struct {
	PlanKey        string `json:"planKey"`
	TriggerReason  string `json:"triggerReason,omitempty"`
	BuildNumber    int    `json:"buildNumber,omitempty"`
	BuildResultKey string `json:"buildResultKey,omitempty"`
	Link           Link   `json:"link,omitempty"`
}

type queueEnvelope struct {
	QueuedBuilds struct {
		Size        int           `json:"size"`
		MaxResult   int           `json:"max-result"`
		StartIndex  int           `json:"start-index"`
		QueuedBuild []QueuedBuild `json:"queuedBuild"`
	} `json:"queuedBuilds"`
}

// ListQueue returns currently queued builds. Results is always non-nil so
// JSON output renders `[]` rather than `null` when the queue is empty.
func (c *Client) ListQueue(ctx context.Context, opts PageOpts) (Page[QueuedBuild], error) {
	var env queueEnvelope
	if err := c.Do(ctx, http.MethodGet, "/api/latest/queue", opts.Values(), nil, &env); err != nil {
		return Page[QueuedBuild]{}, err
	}
	results := env.QueuedBuilds.QueuedBuild
	if results == nil {
		results = []QueuedBuild{}
	}
	return Page[QueuedBuild]{
		Results:    results,
		Size:       env.QueuedBuilds.Size,
		MaxResult:  env.QueuedBuilds.MaxResult,
		StartIndex: env.QueuedBuilds.StartIndex,
	}, nil
}

// TriggerBuild enqueues a build of the plan, returning the queued entry.
// `variables` are passed as bamboo.variable.<key> query params.
func (c *Client) TriggerBuild(ctx context.Context, planKey string, variables map[string]string) (*QueuedBuild, error) {
	q := url.Values{}
	for k, v := range variables {
		q.Set("bamboo.variable."+k, v)
	}
	var qb QueuedBuild
	p := fmt.Sprintf("/api/latest/queue/%s", url.PathEscape(planKey))
	if err := c.Do(ctx, http.MethodPost, p, q, nil, &qb); err != nil {
		return nil, err
	}
	return &qb, nil
}

// StopBuild cancels a queued or in-progress build. `buildResultKey` is `PROJ-PLAN-<n>`.
//
// NOTE: calling this on an already-finished build returns 4xx. That's safe to
// treat as a soft-fail when used as best-effort cleanup (e.g. after a build
// that may or may not have finished by the time the caller stops it).
func (c *Client) StopBuild(ctx context.Context, buildResultKey string) error {
	p := fmt.Sprintf("/api/latest/queue/%s", url.PathEscape(buildResultKey))
	return c.Do(ctx, http.MethodDelete, p, nil, nil, nil)
}

// ContinueBuild resumes a stopped build.
func (c *Client) ContinueBuild(ctx context.Context, buildResultKey string) error {
	p := fmt.Sprintf("/api/latest/queue/%s", url.PathEscape(buildResultKey))
	return c.Do(ctx, http.MethodPut, p, nil, nil, nil)
}

// BuildStatus is the slim response from /result/status/<key>.
type BuildStatus struct {
	Message  string `json:"message,omitempty"`
	Finished bool   `json:"finished"`
	Progress *struct {
		PercentageCompleted float64 `json:"percentageCompleted,omitempty"`
		PrettyTimeRemaining string  `json:"prettyTimeRemaining,omitempty"`
	} `json:"progress,omitempty"`
}

// GetBuildStatus polls live progress of an in-progress build.
//
// NOTE: this endpoint returns 404 once the build has *finished*. Callers
// polling for completion must NOT use this as the primary signal — instead,
// poll GetBuild and inspect `LifeCycleState` (becomes "Finished" when done).
// The 404 here is normal Bamboo behaviour, not an error in our client.
func (c *Client) GetBuildStatus(ctx context.Context, buildResultKey string) (*BuildStatus, error) {
	var s BuildStatus
	p := fmt.Sprintf("/api/latest/result/status/%s", url.PathEscape(buildResultKey))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// EntityKey is the nested key under PlanResultKey ({"key":"PROJ-PLAN"}).
type EntityKey struct {
	Key string `json:"key,omitempty"`
}

// PlanResultKey identifies a specific build result.
type PlanResultKey struct {
	Key          string    `json:"key,omitempty"`
	EntityKey    EntityKey `json:"entityKey,omitempty"`
	ResultNumber int       `json:"resultNumber,omitempty"`
}

// BuildResult is a single build outcome.
type BuildResult struct {
	Key                 string        `json:"key"`
	BuildNumber         int           `json:"buildNumber"`
	PlanKey             string        `json:"planKey,omitempty"`
	State               string        `json:"state,omitempty"`
	LifeCycleState      string        `json:"lifeCycleState,omitempty"`
	BuildState          string        `json:"buildState,omitempty"`
	BuildStartedTime    string        `json:"buildStartedTime,omitempty"`
	BuildCompletedTime  string        `json:"buildCompletedTime,omitempty"`
	BuildDurationInSec  int           `json:"buildDurationInSeconds,omitempty"`
	PrettyBuildDuration string        `json:"prettyBuildDuration,omitempty"`
	BuildReason         string        `json:"buildReason,omitempty"`
	Successful          bool          `json:"successful,omitempty"`
	PlanResultKey       PlanResultKey `json:"planResultKey,omitempty"`
}

type resultsEnvelope struct {
	Results struct {
		Size       int           `json:"size"`
		MaxResult  int           `json:"max-result"`
		StartIndex int           `json:"start-index"`
		Result     []BuildResult `json:"result"`
	} `json:"results"`
}

// GetBuild fetches a single build result by key (e.g. PROJ-PLAN-42).
func (c *Client) GetBuild(ctx context.Context, key string) (*BuildResult, error) {
	var br BuildResult
	p := fmt.Sprintf("/api/latest/result/%s", url.PathEscape(key))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &br); err != nil {
		return nil, err
	}
	return &br, nil
}

// BuildHistory returns the build history for a plan.
func (c *Client) BuildHistory(ctx context.Context, planKey string, opts PageOpts) (Page[BuildResult], error) {
	var env resultsEnvelope
	p := fmt.Sprintf("/api/latest/result/%s", url.PathEscape(planKey))
	if err := c.Do(ctx, http.MethodGet, p, opts.Values(), nil, &env); err != nil {
		return Page[BuildResult]{}, err
	}
	return Page[BuildResult]{
		Results:    env.Results.Result,
		Size:       env.Results.Size,
		MaxResult:  env.Results.MaxResult,
		StartIndex: env.Results.StartIndex,
	}, nil
}

// LatestBuilds returns the latest build across all plans the user can see.
func (c *Client) LatestBuilds(ctx context.Context, opts PageOpts) (Page[BuildResult], error) {
	var env resultsEnvelope
	if err := c.Do(ctx, http.MethodGet, "/api/latest/result", opts.Values(), nil, &env); err != nil {
		return Page[BuildResult]{}, err
	}
	return Page[BuildResult]{
		Results:    env.Results.Result,
		Size:       env.Results.Size,
		MaxResult:  env.Results.MaxResult,
		StartIndex: env.Results.StartIndex,
	}, nil
}

// Build logs -------------------------------------------------------------

// BuildLog pairs one job's log output with the job key it came from. A multi-
// job build returns one BuildLog per job in stage order.
type BuildLog struct {
	JobKey string `json:"jobKey"`
	Log    string `json:"log"`
}

// jobResultEnvelope is the inner shape under
//
//	/result/{key}?expand=stages.stage.results.result
//
// We only care about the result keys, but the structure is nested 4 deep.
type jobResultEnvelope struct {
	Stages struct {
		Stage []struct {
			Results struct {
				Result []struct {
					Key         string `json:"key"`
					BuildNumber int    `json:"buildNumber"`
				} `json:"result"`
			} `json:"results"`
		} `json:"stage"`
	} `json:"stages"`
}

// GetBuildLog fetches the plain-text log for every job in a build.
//
// Implementation notes:
//   - Job keys are discovered via /result/{buildKey}?expand=stages.stage.results.result.
//   - The per-job download URL is /download/{jobResultKey-without-build-suffix}/build_logs/{jobResultKey}.log
//     e.g. /download/PROJ-A-JOB1/build_logs/PROJ-A-JOB1-42.log
//   - Bamboo 8.2.4 serves /download/* via session-cookie auth, NOT Bearer PAT.
//     Bearer requests get HTTP 200 with an HTML login page. We detect this
//     (content-type text/html OR body starts with "<!DOCTYPE") and return a
//     typed session_auth_required error so callers see a clear message
//     instead of an unparseable HTML blob.
func (c *Client) GetBuildLog(ctx context.Context, buildResultKey string) ([]BuildLog, error) {
	q := url.Values{}
	q.Set("expand", "stages.stage.results.result")
	var env jobResultEnvelope
	rp := fmt.Sprintf("/api/latest/result/%s", url.PathEscape(buildResultKey))
	if err := c.Do(ctx, http.MethodGet, rp, q, nil, &env); err != nil {
		return nil, err
	}

	var out []BuildLog
	for _, stage := range env.Stages.Stage {
		for _, result := range stage.Results.Result {
			if result.Key == "" {
				continue
			}
			// Job key = result key without the "-<buildNumber>" suffix.
			// E.g. "PROJ-A-JOB1-42" -> "PROJ-A-JOB1".
			jobKey := result.Key
			if idx := strings.LastIndex(jobKey, "-"); idx > 0 {
				jobKey = jobKey[:idx]
			}
			path := fmt.Sprintf("/download/%s/build_logs/%s.log",
				url.PathEscape(jobKey), url.PathEscape(result.Key))
			ct, body, err := c.doDownload(ctx, path)
			if err != nil {
				return nil, err
			}
			if isHTMLLoginRedirect(ct, body) {
				return nil, &fail.Error{
					Code: "session_auth_required",
					Message: "Bamboo build logs require session-cookie auth on this server; " +
						"Personal Access Tokens are not accepted on /download/* paths. " +
						"See docs/API_COVERAGE.md for details.",
					Exit: fail.ExitAuth,
				}
			}
			out = append(out, BuildLog{JobKey: result.Key, Log: string(body)})
		}
	}
	if out == nil {
		out = []BuildLog{}
	}
	return out, nil
}

func isHTMLLoginRedirect(contentType string, body []byte) bool {
	if strings.HasPrefix(strings.ToLower(contentType), "text/html") {
		return true
	}
	// Some Bamboo versions return the HTML body without setting content-type.
	// Sniff the first bytes.
	trimmed := strings.TrimSpace(string(body))
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html")
}

// Build comments ---------------------------------------------------------

// BuildComment is a comment on a build result. Bamboo Server uses ISO-8601
// timestamp strings under `creationDate`/`modificationDate` — older docs that
// say `date` are stale (verified against 8.2.4).
type BuildComment struct {
	ID               int64  `json:"id"`
	Author           string `json:"author,omitempty"`
	Content          string `json:"content"`
	CreationDate     string `json:"creationDate,omitempty"`
	ModificationDate string `json:"modificationDate,omitempty"`
}

type commentsEnvelope struct {
	Comments struct {
		Size       int            `json:"size"`
		StartIndex int            `json:"start-index"`
		MaxResult  int            `json:"max-result"`
		Comment    []BuildComment `json:"comment"`
	} `json:"comments"`
}

// ListBuildComments returns all comments on a build.
//
// IMPORTANT: Without the `expand=comments.comment` query parameter, Bamboo's
// list endpoint returns only id+author for each comment — the content, dates,
// etc. are omitted. We always pass the expand to give callers a useful payload.
func (c *Client) ListBuildComments(ctx context.Context, buildKey string) ([]BuildComment, error) {
	q := url.Values{}
	q.Set("expand", "comments.comment")
	var env commentsEnvelope
	p := fmt.Sprintf("/api/latest/result/%s/comment", url.PathEscape(buildKey))
	if err := c.Do(ctx, http.MethodGet, p, q, nil, &env); err != nil {
		return nil, err
	}
	out := env.Comments.Comment
	if out == nil {
		out = []BuildComment{}
	}
	return out, nil
}

// AddBuildComment adds a comment to a build. Bamboo Server 8.2.4 returns
// 204 No Content on success (so we cannot return the created entity here);
// callers should re-list comments if they need the new comment's id.
func (c *Client) AddBuildComment(ctx context.Context, buildKey, content string) error {
	body := map[string]string{"content": content}
	p := fmt.Sprintf("/api/latest/result/%s/comment", url.PathEscape(buildKey))
	return c.Do(ctx, http.MethodPost, p, nil, body, nil)
}

func (c *Client) DeleteBuildComment(ctx context.Context, buildKey string, commentID int64) error {
	p := fmt.Sprintf("/api/latest/result/%s/comment/%d", url.PathEscape(buildKey), commentID)
	return c.Do(ctx, http.MethodDelete, p, nil, nil, nil)
}

// Build labels -----------------------------------------------------------

type BuildLabel struct {
	Name string `json:"name"`
}

type labelsEnvelope struct {
	Labels struct {
		Size       int          `json:"size"`
		StartIndex int          `json:"start-index"`
		MaxResult  int          `json:"max-result"`
		Label      []BuildLabel `json:"label"`
	} `json:"labels"`
}

func (c *Client) ListBuildLabels(ctx context.Context, buildKey string) ([]BuildLabel, error) {
	var env labelsEnvelope
	p := fmt.Sprintf("/api/latest/result/%s/label", url.PathEscape(buildKey))
	if err := c.Do(ctx, http.MethodGet, p, nil, nil, &env); err != nil {
		return nil, err
	}
	return env.Labels.Label, nil
}

func (c *Client) AddBuildLabel(ctx context.Context, buildKey, label string) error {
	body := BuildLabel{Name: label}
	p := fmt.Sprintf("/api/latest/result/%s/label", url.PathEscape(buildKey))
	return c.Do(ctx, http.MethodPost, p, nil, body, nil)
}

func (c *Client) DeleteBuildLabel(ctx context.Context, buildKey, label string) error {
	p := fmt.Sprintf("/api/latest/result/%s/label/%s", url.PathEscape(buildKey), url.PathEscape(label))
	return c.Do(ctx, http.MethodDelete, p, nil, nil, nil)
}
