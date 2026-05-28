package api

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rahadiangg/bbx/internal/fail"
)

func TestNewOptionsValidation(t *testing.T) {
	t.Parallel()
	if _, err := New(Options{}); err == nil {
		t.Fatal("expected error when BaseURL missing")
	}
	if _, err := New(Options{BaseURL: "https://x"}); err == nil {
		t.Fatal("expected error when Token missing")
	}
	c, err := New(Options{BaseURL: "https://bamboo.example.com/", Token: "t"})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !strings.HasPrefix(c.BaseURL(), "https://bamboo.example.com") {
		t.Fatalf("BaseURL %q", c.BaseURL())
	}
	if c.httpClient.Timeout == 0 {
		t.Fatalf("expected non-zero default timeout")
	}
}

func TestNewCustomTimeout(t *testing.T) {
	t.Parallel()
	c, err := New(Options{BaseURL: "https://x", Token: "t", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient.Timeout != 5*time.Second {
		t.Fatalf("timeout = %v", c.httpClient.Timeout)
	}
}

func TestDoSendsBearerAndDecodes(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("GET", "/rest/api/latest/plan", 200, `{"plans":{"size":0,"max-result":0,"start-index":0,"plan":[]}}`)
	c := newTestClient(t, fb)

	var out planEnvelope
	if err := c.Do(context.Background(), "GET", "/api/latest/plan", nil, nil, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if rec.AuthHeader != "Bearer test-token" {
		t.Errorf("Authorization = %q", rec.AuthHeader)
	}
	if rec.Accept != "application/json" {
		t.Errorf("Accept = %q", rec.Accept)
	}
	if !strings.HasPrefix(rec.UserAgent, "bbx/") {
		t.Errorf("User-Agent = %q", rec.UserAgent)
	}
}

func TestDoEncodesBodyAndSetsContentType(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("POST", "/rest/api/latest/queue/PROJ-PLAN", 200, `{"planKey":"PROJ-PLAN"}`)
	c := newTestClient(t, fb)

	body := map[string]string{"k": "v"}
	var out map[string]string
	if err := c.Do(context.Background(), "POST", "/api/latest/queue/PROJ-PLAN", nil, body, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if rec.ContentType != "application/json" {
		t.Errorf("Content-Type = %q", rec.ContentType)
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body, &got); err != nil {
		t.Fatalf("body: %v", err)
	}
	if got["k"] != "v" {
		t.Errorf("body = %v", got)
	}
}

func TestDoQueryStringEncoded(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	rec := fb.expect("GET", "/rest/api/latest/plan", 200, `{}`)
	c := newTestClient(t, fb)

	q := url.Values{"start-index": []string{"50"}, "max-results": []string{"100"}}
	if err := c.Do(context.Background(), "GET", "/api/latest/plan", q, nil, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if !strings.Contains(rec.RawQuery, "start-index=50") || !strings.Contains(rec.RawQuery, "max-results=100") {
		t.Errorf("query = %q", rec.RawQuery)
	}
}

func TestDoNoContentSuccess(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("DELETE", "/rest/api/latest/plan/PROJ-PLAN", 204, "")
	c := newTestClient(t, fb)
	if err := c.Do(context.Background(), "DELETE", "/api/latest/plan/PROJ-PLAN", nil, nil, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestDoReturnsTypedErrorOn401(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/currentUser", 401, `{"message":"You are not authenticated"}`)
	c := newTestClient(t, fb)

	err := c.Do(context.Background(), "GET", "/api/latest/currentUser", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var fe *fail.Error
	if !errAs(err, &fe) {
		t.Fatalf("err is %T %v, want *fail.Error", err, err)
	}
	if fe.HTTPStatus != 401 {
		t.Errorf("HTTPStatus = %d", fe.HTTPStatus)
	}
	if fe.Exit != fail.ExitAuth {
		t.Errorf("Exit = %d, want %d", fe.Exit, fail.ExitAuth)
	}
	if !strings.Contains(fe.Message, "not authenticated") {
		t.Errorf("Message = %q", fe.Message)
	}
}

func TestDoErrorOn404UsageExit(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan/MISSING", 404, `{"message":"Plan not found"}`)
	c := newTestClient(t, fb)
	err := c.Do(context.Background(), "GET", "/api/latest/plan/MISSING", nil, nil, nil)
	var fe *fail.Error
	if !errAs(err, &fe) {
		t.Fatalf("err is %v", err)
	}
	if fe.Exit != fail.ExitUsage {
		t.Errorf("Exit = %d", fe.Exit)
	}
}

func TestDoErrorOn500GenericExit(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan", 500, `{"message":"oops"}`)
	c := newTestClient(t, fb)
	err := c.Do(context.Background(), "GET", "/api/latest/plan", nil, nil, nil)
	var fe *fail.Error
	if !errAs(err, &fe) {
		t.Fatalf("err is %v", err)
	}
	if fe.Exit != fail.ExitGeneric {
		t.Errorf("Exit = %d", fe.Exit)
	}
}

func TestDoErrorFallbackToBodySnippet(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan", 500, "<html>boom</html>")
	c := newTestClient(t, fb)
	err := c.Do(context.Background(), "GET", "/api/latest/plan", nil, nil, nil)
	var fe *fail.Error
	if !errAs(err, &fe) {
		t.Fatalf("err is %v", err)
	}
	if !strings.Contains(fe.Message, "boom") {
		t.Errorf("Message = %q", fe.Message)
	}
}

func TestDoDecodeError(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/plan", 200, `{not json`)
	c := newTestClient(t, fb)
	var out planEnvelope
	if err := c.Do(context.Background(), "GET", "/api/latest/plan", nil, nil, &out); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestResolvePrependsRest(t *testing.T) {
	t.Parallel()
	c, _ := New(Options{BaseURL: "https://bamboo.example.com", Token: "t"})
	u := c.resolve("/api/latest/plan")
	if u.Path != "/rest/api/latest/plan" {
		t.Fatalf("resolve = %q", u.Path)
	}
	u2 := c.resolve("api/latest/plan") // no leading slash
	if u2.Path != "/rest/api/latest/plan" {
		t.Fatalf("resolve = %q", u2.Path)
	}
	u3 := c.resolve("/rest/agent/list")
	if u3.Path != "/rest/agent/list" {
		t.Fatalf("resolve = %q", u3.Path)
	}
}

func TestResolveWithBaseURLPath(t *testing.T) {
	t.Parallel()
	c, _ := New(Options{BaseURL: "https://bamboo.example.com/bamboo/", Token: "t"})
	u := c.resolve("/api/latest/plan")
	if u.Path != "/bamboo/rest/api/latest/plan" {
		t.Fatalf("resolve = %q", u.Path)
	}
}

func TestGetRaw(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/raw", 200, "hello world")
	c := newTestClient(t, fb)
	b, err := c.GetRaw(context.Background(), "/api/latest/raw", nil)
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if string(b) != "hello world" {
		t.Fatalf("got %q", b)
	}
}

func TestGetRawErrorPropagates(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/raw", 401, `{"message":"no auth"}`)
	c := newTestClient(t, fb)
	_, err := c.GetRaw(context.Background(), "/api/latest/raw", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var fe *fail.Error
	if !errAs(err, &fe) || fe.Exit != fail.ExitAuth {
		t.Fatalf("err = %v", err)
	}
}
