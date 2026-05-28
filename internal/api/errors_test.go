package api

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

// errAs is a tiny errors.As helper shared across api tests.
func errAs[T error](err error, target *T) bool { return errors.As(err, target) }

func TestAPIErrorString(t *testing.T) {
	t.Parallel()
	e := &APIError{Status: 400, Code: "Bad Request", Message: "x"}
	if !strings.Contains(e.Error(), "400") || !strings.Contains(e.Error(), "Bad Request") {
		t.Fatalf("Error = %q", e.Error())
	}
	e2 := &APIError{Status: 422, Code: "Unprocessable", Message: "bad", Errors: []string{"f1", "f2"}}
	if !strings.Contains(e2.Error(), "f1") || !strings.Contains(e2.Error(), "f2") {
		t.Fatalf("Error = %q", e2.Error())
	}
}

func TestParseErrorFallsBackToErrorsArray(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("POST", "/rest/api/latest/queue/deployment", 400, `{"errors":["Please supply a deploymentProjectId"],"fieldErrors":{}}`)
	c := newTestClient(t, fb)
	_, err := c.TriggerDeployment(t.Context(), 0, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Please supply a deploymentProjectId") {
		t.Fatalf("err message lost: %v", err)
	}
}

func TestAPIErrorToFailExitMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status int
		exit   int
	}{
		{http.StatusUnauthorized, fail.ExitAuth},
		{http.StatusForbidden, fail.ExitAuth},
		{http.StatusBadRequest, fail.ExitUsage},
		{http.StatusNotFound, fail.ExitUsage},
		{http.StatusConflict, fail.ExitUsage},
		{http.StatusInternalServerError, fail.ExitGeneric},
		{http.StatusBadGateway, fail.ExitGeneric},
	}
	for _, c := range cases {
		c := c
		t.Run(http.StatusText(c.status), func(t *testing.T) {
			e := &APIError{Status: c.status, Code: "x", Message: "y"}
			fe := e.ToFail()
			if fe.Exit != c.exit {
				t.Fatalf("status %d -> exit %d, want %d", c.status, fe.Exit, c.exit)
			}
			if fe.HTTPStatus != c.status {
				t.Fatalf("HTTPStatus = %d", fe.HTTPStatus)
			}
		})
	}
}
