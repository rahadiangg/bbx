package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rahadiangg/bbx/cmd/cmdctx"
	"github.com/rahadiangg/bbx/internal/fail"
	"github.com/rahadiangg/bbx/internal/output"
)

// recordedReq mirrors the API package's helper for cmd-level assertions.
type recordedReq struct {
	Method   string
	Path     string
	RawQuery string
	Body     []byte
	Auth     string
}

type fakeRoute struct {
	Status  int
	Body    string
	Capture *recordedReq
}

// fakeBamboo is a tiny httptest server used by cmd integration tests.
type fakeBamboo struct {
	t      *testing.T
	srv    *httptest.Server
	mu     sync.Mutex
	routes map[string]*fakeRoute
}

func newFakeBamboo(t *testing.T) *fakeBamboo {
	t.Helper()
	fb := &fakeBamboo{t: t, routes: map[string]*fakeRoute{}}
	fb.srv = httptest.NewServer(http.HandlerFunc(fb.handle))
	t.Cleanup(fb.srv.Close)
	return fb
}

func (f *fakeBamboo) URL() string { return f.srv.URL }

func (f *fakeBamboo) expect(method, path string, status int, body string) *recordedReq {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec := &recordedReq{}
	f.routes[method+" "+path] = &fakeRoute{Status: status, Body: body, Capture: rec}
	return rec
}

func (f *fakeBamboo) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	rt, ok := f.routes[r.Method+" "+r.URL.Path]
	f.mu.Unlock()
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"unknown route `+r.Method+" "+r.URL.Path+`"}`)
		return
	}
	body, _ := io.ReadAll(r.Body)
	*rt.Capture = recordedReq{
		Method:   r.Method,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
		Body:     body,
		Auth:     r.Header.Get("Authorization"),
	}
	if rt.Body != "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(rt.Status)
	if rt.Body != "" {
		_, _ = io.WriteString(w, rt.Body)
	}
}

// envResult is the outcome of a single test invocation.
type envResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// runCmdEnv wraps writeConfig + runCmd: writes a minimal config pointing at fb,
// then runs the root command with the given args. BBX_AGENT_MODE is forced on
// so the runner always emits JSON for deterministic assertions.
func runCmdEnv(t *testing.T, fb *fakeBamboo, args ...string) envResult {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeConfigFile(t, cfgPath, fb.URL(), "test-token")
	t.Setenv("BBX_CONFIG", cfgPath)
	t.Setenv("BBX_AGENT_MODE", "1")
	t.Setenv("BBX_TOKEN", "") // don't let CI env leak
	return runCmd(t, args...)
}

// runCmdNoConfig runs the root command without writing a config file. Used to
// test error paths.
func runCmdNoConfig(t *testing.T, args ...string) envResult {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("BBX_CONFIG", filepath.Join(dir, "absent.yaml"))
	t.Setenv("BBX_AGENT_MODE", "1")
	t.Setenv("BBX_TOKEN", "")
	return runCmd(t, args...)
}

// runCmd invokes a fresh root command. It captures os.Stdout/os.Stderr via
// pipes and returns the result.
func runCmd(t *testing.T, args ...string) envResult {
	t.Helper()

	// Reset the cmdctx singleton so each invocation starts clean. The simplest
	// way is to call cmdctx.Set with a zero value; the PersistentPreRunE will
	// overwrite the relevant fields.
	cmdctx.Set(cmdctx.Globals{})

	stdoutR, stdoutW, _ := os.Pipe()
	stderrR, stderrW, _ := os.Pipe()
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutW, stderrW
	defer func() {
		os.Stdout, os.Stderr = origOut, origErr
	}()

	root := New()
	root.SetArgs(args)

	err := root.Execute()
	if err != nil {
		// Mirror cmd.Execute(): emit a structured error before unwinding.
		output.PrintError(cmdctx.G().Format, err)
	}
	_ = stdoutW.Close()
	_ = stderrW.Close()
	outBytes, _ := io.ReadAll(stdoutR)
	errBytes, _ := io.ReadAll(stderrR)

	exit := fail.ExitOK
	if err != nil {
		// Mirror cmd.Execute() exit-code logic: cobra-level errors (bad arg
		// count, unknown flag, unknown command) map to ExitUsage; *fail.Error
		// carries its own exit.
		exit = exitCodeFor(err)
	}
	return envResult{
		Stdout:   string(outBytes),
		Stderr:   string(errBytes),
		ExitCode: exit,
		Err:      err,
	}
}

func writeConfigFile(t *testing.T, path, baseURL, token string) {
	t.Helper()
	content := "" +
		"current-context: default\n" +
		"contexts:\n" +
		"  default:\n" +
		"    base-url: " + baseURL + "\n" +
		"    token: " + token + "\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// mustDecodeJSON decodes the stdout buffer into v.
func mustDecodeJSON(t *testing.T, s string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(strings.TrimSpace(s)), v); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, s)
	}
}
