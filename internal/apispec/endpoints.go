// Package apispec carries the authoritative registry of every Bamboo REST
// endpoint bbx calls, plus a lightweight loader + checker that asserts
// each registry entry is present in a Bamboo OpenAPI/swagger spec.
//
// The CI api-compat workflow runs this check against multiple Bamboo
// versions to catch path/method removals between releases before they
// break a user's upgrade.
//
// If you add or remove a bbx API call in `internal/api/*.go`, mirror that
// change here. The test in compat_test.go is the safety net.
package apispec

// Endpoint declares one HTTP call bbx makes against Bamboo. `Path` uses
// Bamboo's exact swagger-style placeholders (composite keys like
// `{projectKey}-{buildKey}` are kept verbatim — Atlassian's spec defines
// them as separate placeholders joined by a literal `-`).
//
// `Excluded` flags endpoints we know are NOT covered by the published
// OpenAPI spec — currently just the `/download/*` log paths, which are
// served outside the /rest namespace entirely.
type Endpoint struct {
	Method   string // uppercase: GET, POST, PUT, DELETE
	Path     string // swagger-relative, e.g. "/api/latest/plan/{projectKey}-{buildKey}"
	Excluded bool   // true => skip in compat check
	Note     string // optional human description, surfaced in diff output
}

// All returns the registry of every endpoint bbx calls. Ordered loosely by
// internal/api file for ease of diffing against the source.
func All() []Endpoint {
	return []Endpoint{
		// info.go
		{Method: "GET", Path: "/api/latest/info", Note: "GetServerInfo"},
		// deployment.go (currentUser is here historically)
		{Method: "GET", Path: "/api/latest/currentUser", Note: "WhoAmI"},

		// plan.go
		{Method: "GET", Path: "/api/latest/plan", Note: "ListPlans"},
		{Method: "GET", Path: "/api/latest/plan/{projectKey}-{buildKey}", Note: "GetPlan"},
		{Method: "POST", Path: "/api/latest/plan/{projectKey}-{buildKey}/enable", Note: "EnablePlan"},
		{Method: "DELETE", Path: "/api/latest/plan/{projectKey}-{buildKey}/enable", Note: "DisablePlan"},
		{Method: "DELETE", Path: "/api/latest/plan/{projectKey}-{buildKey}", Note: "DeletePlan"},
		{Method: "PUT", Path: "/api/latest/clone/{projectKey}-{buildKey}:{toProjectKey}-{toBuildKey}", Note: "ClonePlan"},
		{Method: "GET", Path: "/api/latest/plan/{projectKey}-{buildKey}/branch", Note: "ListPlanBranches"},
		{Method: "GET", Path: "/api/latest/plan/{projectKey}-{buildKey}/branch/{branchName}", Note: "GetPlanBranch"},
		{Method: "PUT", Path: "/api/latest/plan/{projectKey}-{buildKey}/branch/{branchName}", Note: "CreatePlanBranch"},
		{Method: "GET", Path: "/api/latest/plan/{projectKey}-{buildKey}/variables", Note: "ListPlanVariables"},
		{Method: "POST", Path: "/api/latest/plan/{projectKey}-{buildKey}/variables", Note: "AddPlanVariable"},
		{Method: "GET", Path: "/api/latest/plan/{projectKey}-{buildKey}/variables/{variableName}", Note: "GetPlanVariable"},
		{Method: "PUT", Path: "/api/latest/plan/{projectKey}-{buildKey}/variables/{variableName}", Note: "UpdatePlanVariable"},
		{Method: "DELETE", Path: "/api/latest/plan/{projectKey}-{buildKey}/variables/{variableName}", Note: "DeletePlanVariable"},

		// build.go — queue + results
		{Method: "GET", Path: "/api/latest/queue", Note: "ListQueue"},
		{Method: "POST", Path: "/api/latest/queue/{projectKey}-{buildKey}", Note: "TriggerBuild"},
		{Method: "DELETE", Path: "/api/latest/queue/{projectKey}-{buildKey}-{buildNumber}", Note: "StopBuild"},
		{Method: "PUT", Path: "/api/latest/queue/{projectKey}-{buildKey}-{buildNumber}", Note: "ContinueBuild"},
		{Method: "GET", Path: "/api/latest/result/status/{projectKey}-{buildKey}-{buildNumber}", Note: "GetBuildStatus"},
		{Method: "GET", Path: "/api/latest/result/{projectKey}-{buildKey}-{buildNumber}", Note: "GetBuild"},
		{Method: "GET", Path: "/api/latest/result/{projectKey}-{buildKey}", Note: "BuildHistory"},
		{Method: "GET", Path: "/api/latest/result", Note: "LatestBuilds"},
		{Method: "GET", Path: "/api/latest/result/{projectKey}-{buildKey}-{buildNumber}/comment", Note: "ListBuildComments"},
		{Method: "POST", Path: "/api/latest/result/{projectKey}-{buildKey}-{buildNumber}/comment", Note: "AddBuildComment"},
		{Method: "DELETE", Path: "/api/latest/result/{projectKey}-{buildKey}-{buildNumber}/comment/{commentId}", Note: "DeleteBuildComment"},
		{Method: "GET", Path: "/api/latest/result/{projectKey}-{buildKey}-{buildNumber}/label", Note: "ListBuildLabels"},
		{Method: "POST", Path: "/api/latest/result/{projectKey}-{buildKey}-{buildNumber}/label", Note: "AddBuildLabel"},
		{Method: "DELETE", Path: "/api/latest/result/{projectKey}-{buildKey}-{buildNumber}/label/{labelName}", Note: "DeleteBuildLabel"},

		// build.go — log fetch (NOT in OpenAPI; served outside /rest)
		{
			Method: "GET", Path: "/download/{jobKey}/build_logs/{jobKey}-{buildNumber}.log",
			Excluded: true, Note: "GetBuildLog (non-REST, /download/* — session-cookie auth on 8.x)",
		},

		// deployment.go — runtime
		{Method: "GET", Path: "/api/latest/queue/deployment", Note: "ListDeploymentQueue"},
		{Method: "POST", Path: "/api/latest/queue/deployment", Note: "TriggerDeployment"},
		{Method: "DELETE", Path: "/api/latest/queue/deployment/{deploymentResultId}", Note: "CancelDeployment"},
		{Method: "GET", Path: "/api/latest/deploy/result/{deploymentResultId}", Note: "GetDeploymentResult"},
		{Method: "GET", Path: "/api/latest/deploy/preview/version", Note: "PreviewDeploymentVersion"},

		// plan.go — config extraction
		{Method: "GET", Path: "/api/latest/plan/{projectKey}-{buildKey}/specs", Note: "GetPlanSpec"},
		{Method: "GET", Path: "/api/latest/plan/{projectKey}-{buildKey}/artifact", Note: "ListPlanArtifacts"},
		{Method: "GET", Path: "/api/latest/plan/{projectKey}-{buildKey}/vcsBranches", Note: "ListPlanVCSBranches"},
		// GetPlanConfig hits the same /plan/{projectKey}-{buildKey} path as GetPlan,
		// just with an `expand` query. No separate registry entry needed.

		// project.go — new
		{Method: "GET", Path: "/api/latest/project/{projectKey}", Note: "GetProject"},
		{Method: "GET", Path: "/api/latest/project/{projectKey}/specs", Note: "GetProjectSpec"},
		{Method: "GET", Path: "/api/latest/project/{projectKey}/variables", Note: "ListProjectVariables"},
		{Method: "GET", Path: "/api/latest/project/{projectKey}/variable/{variableName}", Note: "GetProjectVariable"},
		{Method: "GET", Path: "/api/latest/project/{projectKey}/repository", Note: "ListProjectRepositories"},

		// deployment.go — config extraction (project + environment + versions)
		{Method: "GET", Path: "/api/latest/deploy/project/all", Note: "ListDeploymentProjects"},
		{Method: "GET", Path: "/api/latest/deploy/project/forPlan", Note: "ListDeploymentProjectsForPlan"},
		{Method: "GET", Path: "/api/latest/deploy/project/{deploymentProjectId}", Note: "GetDeploymentProject"},
		{Method: "GET", Path: "/api/latest/deploy/project/{deploymentProjectId}/specs", Note: "GetDeploymentProjectSpec"},
		{Method: "GET", Path: "/api/latest/deploy/project/{deploymentProjectId}/repository", Note: "ListDeploymentProjectRepositories"},
		{Method: "GET", Path: "/api/latest/deploy/project/{deploymentProjectId}/versions", Note: "ListDeploymentVersions"},
		{Method: "GET", Path: "/api/latest/deploy/environment/{environmentId}", Note: "GetDeploymentEnvironment"},
		{Method: "GET", Path: "/api/latest/deploy/environment/{environmentId}/variables", Note: "ListEnvironmentVariables"},
		{Method: "GET", Path: "/api/latest/deploy/environment/{environmentId}/requirement", Note: "ListEnvironmentRequirements"},
		{Method: "GET", Path: "/api/latest/deploy/environment/{environmentId}/agent-assignment", Note: "ListEnvironmentAgentAssignments"},
	}
}
