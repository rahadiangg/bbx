package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// recordedReq captures the salient fields of an inbound request for assertions.
type recordedReq struct {
	Method      string
	Path        string
	RawQuery    string
	Body        []byte
	AuthHeader  string
	Accept      string
	UserAgent   string
	ContentType string
}

// route maps a "METHOD /path" pair to a handler-like description.
type route struct {
	Status  int
	Body    string
	Capture *recordedReq
}

// fakeBamboo spins up a small httptest server that records the most recent
// request and returns a configured response for the requested method+path.
// Routes match exactly on "METHOD /path"; the route returned for an unmatched
// request is a 404 with a JSON payload Bamboo would typically emit.
type fakeBamboo struct {
	t      *testing.T
	server *httptest.Server
	routes map[string]*route
}

func newFakeBamboo(t *testing.T) *fakeBamboo {
	t.Helper()
	fb := &fakeBamboo{t: t, routes: map[string]*route{}}
	fb.server = httptest.NewServer(http.HandlerFunc(fb.handle))
	t.Cleanup(fb.server.Close)
	return fb
}

func (f *fakeBamboo) URL() string { return f.server.URL }

// expect registers a route. Capture is populated when the route is hit.
func (f *fakeBamboo) expect(method, path string, status int, body string) *recordedReq {
	rec := &recordedReq{}
	f.routes[method+" "+path] = &route{Status: status, Body: body, Capture: rec}
	return rec
}

func (f *fakeBamboo) handle(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	rt, ok := f.routes[key]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"unknown route `+key+`"}`)
		return
	}
	body, _ := io.ReadAll(r.Body)
	*rt.Capture = recordedReq{
		Method:      r.Method,
		Path:        r.URL.Path,
		RawQuery:    r.URL.RawQuery,
		Body:        body,
		AuthHeader:  r.Header.Get("Authorization"),
		Accept:      r.Header.Get("Accept"),
		UserAgent:   r.Header.Get("User-Agent"),
		ContentType: r.Header.Get("Content-Type"),
	}
	if rt.Body != "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(rt.Status)
	if rt.Body != "" {
		_, _ = io.WriteString(w, rt.Body)
	}
}

// newTestClient returns a Client configured against fb.
func newTestClient(t *testing.T, fb *fakeBamboo) *Client {
	t.Helper()
	c, err := New(Options{BaseURL: fb.URL(), Token: "test-token"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}
