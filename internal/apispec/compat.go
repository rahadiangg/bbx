package apispec

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Spec is the slim view of a Bamboo OpenAPI document. We only care about the
// `paths` map (path -> method -> operation); full schema validation is out
// of scope. Both Swagger v2 and OpenAPI v3 use the same path structure.
type Spec struct {
	Source      string                         // URL or file path it came from
	Format      string                         // "openapi-3.x" or "swagger-2.0"
	BasePath    string                         // v2 may set "basePath"; we strip it before matching
	PathMethods map[string]map[string]struct{} // path -> set of uppercase methods
}

// rawSpec captures just enough of the JSON to extract the info we need.
type rawSpec struct {
	OpenAPI  string                            `json:"openapi"`
	Swagger  string                            `json:"swagger"`
	BasePath string                            `json:"basePath"`
	Paths    map[string]map[string]interface{} `json:"paths"`
}

// LoadSpec loads an OpenAPI/swagger JSON from either an HTTP/HTTPS URL or a
// local file path. Auto-detects by scheme.
func LoadSpec(loc string) (*Spec, error) {
	var body []byte
	switch {
	case strings.HasPrefix(loc, "http://"), strings.HasPrefix(loc, "https://"):
		req, err := http.NewRequest(http.MethodGet, loc, nil)
		if err != nil {
			return nil, err
		}
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", loc, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("fetch %s: HTTP %d", loc, resp.StatusCode)
		}
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", loc, err)
		}
	default:
		var err error
		body, err = os.ReadFile(loc)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", loc, err)
		}
	}

	var raw rawSpec
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", loc, err)
	}

	format := ""
	switch {
	case raw.OpenAPI != "":
		format = "openapi-" + raw.OpenAPI
	case raw.Swagger != "":
		format = "swagger-" + raw.Swagger
	default:
		return nil, fmt.Errorf("%s: neither `openapi` nor `swagger` field present", loc)
	}
	if len(raw.Paths) == 0 {
		return nil, fmt.Errorf("%s: spec has no `paths`", loc)
	}

	spec := &Spec{
		Source:      loc,
		Format:      format,
		BasePath:    raw.BasePath,
		PathMethods: make(map[string]map[string]struct{}, len(raw.Paths)),
	}
	for p, ops := range raw.Paths {
		methods := make(map[string]struct{}, len(ops))
		for m := range ops {
			methods[strings.ToUpper(m)] = struct{}{}
		}
		spec.PathMethods[p] = methods
	}
	return spec, nil
}

// CheckReport summarises a compat run.
type CheckReport struct {
	SpecSource       string
	SpecFormat       string
	Checked          int
	Excluded         int
	Missing          []Endpoint       // path absent from spec entirely
	MethodMismatches []MethodMismatch // path present but expected method missing
}

// MethodMismatch is an endpoint whose path exists in the spec but with a
// different set of methods than bbx expects.
type MethodMismatch struct {
	Endpoint         Endpoint
	AvailableMethods []string // sorted
}

// OK returns true when there are no incompatibilities to report.
func (r *CheckReport) OK() bool {
	return len(r.Missing) == 0 && len(r.MethodMismatches) == 0
}

// Summary returns a multi-line human-readable description of the report,
// suitable for surfacing in CI logs or test failure messages.
func (r *CheckReport) Summary() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "API compat report\n")
	fmt.Fprintf(&sb, "  spec: %s (%s)\n", r.SpecSource, r.SpecFormat)
	fmt.Fprintf(&sb, "  checked: %d, excluded: %d, missing: %d, method mismatches: %d\n",
		r.Checked, r.Excluded, len(r.Missing), len(r.MethodMismatches))
	if len(r.Missing) > 0 {
		fmt.Fprintf(&sb, "\nMissing endpoints (in bbx, not in spec):\n")
		for _, e := range r.Missing {
			fmt.Fprintf(&sb, "  - %s %s  [%s]\n", e.Method, e.Path, e.Note)
		}
	}
	if len(r.MethodMismatches) > 0 {
		fmt.Fprintf(&sb, "\nMethod mismatches (path exists but method differs):\n")
		for _, m := range r.MethodMismatches {
			fmt.Fprintf(&sb, "  - %s %s  [%s]  (spec has: %s)\n",
				m.Endpoint.Method, m.Endpoint.Path, m.Endpoint.Note, strings.Join(m.AvailableMethods, ", "))
		}
	}
	return sb.String()
}

// Check runs each non-excluded endpoint against the spec.
func Check(spec *Spec, endpoints []Endpoint) CheckReport {
	report := CheckReport{
		SpecSource: spec.Source,
		SpecFormat: spec.Format,
	}
	for _, e := range endpoints {
		if e.Excluded {
			report.Excluded++
			continue
		}
		report.Checked++
		// Strip the spec's basePath from our path if present (v2 quirk).
		want := e.Path
		if spec.BasePath != "" && strings.HasPrefix(want, spec.BasePath) {
			want = strings.TrimPrefix(want, spec.BasePath)
		}
		methods, ok := spec.PathMethods[want]
		if !ok {
			report.Missing = append(report.Missing, e)
			continue
		}
		if _, methodOK := methods[strings.ToUpper(e.Method)]; !methodOK {
			avail := make([]string, 0, len(methods))
			for m := range methods {
				avail = append(avail, m)
			}
			// Stable order for readable diffs
			sortStrings(avail)
			report.MethodMismatches = append(report.MethodMismatches, MethodMismatch{
				Endpoint:         e,
				AvailableMethods: avail,
			})
		}
	}
	return report
}

// sortStrings is a tiny dependency-free sort to keep imports minimal.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
