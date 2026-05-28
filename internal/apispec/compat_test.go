package apispec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAPICompat is the production compat check. It runs bbx's endpoint
// registry against a Bamboo OpenAPI spec.
//
//   - Local dev / unit-test runs: uses the vendored testdata snapshot
//     (Bamboo 12.1.1). Hermetic, fast.
//   - CI matrix runs: set BBX_COMPAT_SWAGGER to each version's URL
//     (https://docs.atlassian.com/atlassian-bamboo/REST/<VERSION>/swagger.json).
//
// The test fails with a structured CheckReport on any missing endpoint
// or method mismatch — that's the signal that an upcoming Bamboo version
// will break bbx and the matter needs a code change before merging.
func TestAPICompat(t *testing.T) {
	src := os.Getenv("BBX_COMPAT_SWAGGER")
	if src == "" {
		src = filepath.Join("testdata", "bamboo-12.1.1-swagger.json")
	}
	spec, err := LoadSpec(src)
	if err != nil {
		t.Fatalf("LoadSpec(%s): %v", src, err)
	}
	t.Logf("loaded %s (%s, %d paths)", spec.Source, spec.Format, len(spec.PathMethods))

	report := Check(spec, All())
	if !report.OK() {
		t.Fatalf("compat failures against %s:\n%s", src, report.Summary())
	}
	t.Logf("\n%s", report.Summary())
}

func TestLoadSpecOpenAPI3(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.json")
	body := `{"openapi":"3.0.1","paths":{"/x":{"get":{}}}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(path)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if !strings.HasPrefix(spec.Format, "openapi-") {
		t.Errorf("Format = %q, want openapi-*", spec.Format)
	}
	if _, ok := spec.PathMethods["/x"]["GET"]; !ok {
		t.Fatalf("expected GET on /x, got %v", spec.PathMethods)
	}
}

func TestLoadSpecSwagger2WithBasePath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.json")
	body := `{"swagger":"2.0","basePath":"/rest","paths":{"/x":{"post":{}}}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := LoadSpec(path)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if !strings.HasPrefix(spec.Format, "swagger-") {
		t.Errorf("Format = %q, want swagger-*", spec.Format)
	}
	if spec.BasePath != "/rest" {
		t.Errorf("BasePath = %q", spec.BasePath)
	}
}

func TestLoadSpecRejectsBadFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for name, body := range map[string]string{
		"neither key": `{"paths":{"/x":{"get":{}}}}`,
		"no paths":    `{"openapi":"3.0.0"}`,
		"invalid json": `not-json`,
	} {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(dir, name+".json")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadSpec(path); err == nil {
				t.Fatalf("expected error for %q", name)
			}
		})
	}
}

func TestCheckReportsMissing(t *testing.T) {
	t.Parallel()
	spec := &Spec{
		PathMethods: map[string]map[string]struct{}{
			"/api/latest/info": {"GET": {}},
		},
	}
	got := Check(spec, []Endpoint{
		{Method: "GET", Path: "/api/latest/info"},
		{Method: "GET", Path: "/api/latest/does-not-exist", Note: "fake"},
	})
	if len(got.Missing) != 1 || got.Missing[0].Path != "/api/latest/does-not-exist" {
		t.Fatalf("Missing = %+v", got.Missing)
	}
	if len(got.MethodMismatches) != 0 {
		t.Errorf("unexpected mismatches: %+v", got.MethodMismatches)
	}
	if got.Checked != 2 {
		t.Errorf("Checked = %d", got.Checked)
	}
}

func TestCheckReportsMethodMismatch(t *testing.T) {
	t.Parallel()
	spec := &Spec{
		PathMethods: map[string]map[string]struct{}{
			"/api/latest/x": {"GET": {}},
		},
	}
	got := Check(spec, []Endpoint{
		{Method: "POST", Path: "/api/latest/x", Note: "expected POST"},
	})
	if len(got.Missing) != 0 {
		t.Fatalf("unexpected Missing: %+v", got.Missing)
	}
	if len(got.MethodMismatches) != 1 {
		t.Fatalf("MethodMismatches = %+v", got.MethodMismatches)
	}
	if got.MethodMismatches[0].AvailableMethods[0] != "GET" {
		t.Errorf("AvailableMethods = %v", got.MethodMismatches[0].AvailableMethods)
	}
}

func TestExcludedEndpointsAreSkipped(t *testing.T) {
	t.Parallel()
	spec := &Spec{PathMethods: map[string]map[string]struct{}{}}
	got := Check(spec, []Endpoint{
		{Method: "GET", Path: "/download/whatever", Excluded: true, Note: "non-REST"},
	})
	if got.Excluded != 1 {
		t.Errorf("Excluded = %d", got.Excluded)
	}
	if len(got.Missing) != 0 {
		t.Errorf("excluded endpoint should not appear in Missing: %+v", got.Missing)
	}
}

func TestReportSummaryHumanReadable(t *testing.T) {
	t.Parallel()
	r := &CheckReport{
		SpecSource: "foo.json", SpecFormat: "openapi-3.0",
		Checked: 3, Excluded: 1,
		Missing: []Endpoint{{Method: "GET", Path: "/x", Note: "X"}},
	}
	s := r.Summary()
	for _, want := range []string{"foo.json", "missing: 1", "GET /x", "[X]"} {
		if !strings.Contains(s, want) {
			t.Errorf("summary missing %q:\n%s", want, s)
		}
	}
}

// TestRegistryIntegrityAgainstVendoredSpec verifies that every entry in
// All() (except Excluded ones) appears in the vendored 12.1.1 spec. This
// makes the registry self-validating during normal `go test` runs.
func TestRegistryIntegrityAgainstVendoredSpec(t *testing.T) {
	spec, err := LoadSpec(filepath.Join("testdata", "bamboo-12.1.1-swagger.json"))
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	report := Check(spec, All())
	if !report.OK() {
		t.Fatalf("registry inconsistent with vendored spec:\n%s", report.Summary())
	}
}
