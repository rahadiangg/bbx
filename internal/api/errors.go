package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rahadiangg/bbx/internal/fail"
)

// APIError is a typed error returned by the Bamboo REST API.
type APIError struct {
	Status  int      `json:"http_status"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
	URL     string   `json:"url,omitempty"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("bamboo api %d %s: %s (%s)", e.Status, e.Code, e.Message, strings.Join(e.Errors, "; "))
	}
	return fmt.Sprintf("bamboo api %d %s: %s", e.Status, e.Code, e.Message)
}

// ToFail returns a fail.Error mirroring the API error with an appropriate exit code.
func (e *APIError) ToFail() *fail.Error {
	exit := fail.ExitGeneric
	switch e.Status {
	case http.StatusUnauthorized, http.StatusForbidden:
		exit = fail.ExitAuth
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict:
		exit = fail.ExitUsage
	}
	return &fail.Error{
		Code:       e.Code,
		Message:    e.Message,
		HTTPStatus: e.Status,
		Exit:       exit,
	}
}

// parseError extracts an APIError from a non-2xx response.
func parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	ae := &APIError{
		Status: resp.StatusCode,
		Code:   http.StatusText(resp.StatusCode),
		URL:    resp.Request.URL.String(),
	}
	// Try to decode Bamboo's error JSON. Shape varies, so we accept a few keys.
	var payload struct {
		Message string   `json:"message"`
		Status  int      `json:"status-code"`
		Errors  []string `json:"errors"`
		Cause   string   `json:"cause"`
		Reason  string   `json:"reason"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil {
		switch {
		case payload.Message != "":
			ae.Message = payload.Message
		case payload.Cause != "":
			ae.Message = payload.Cause
		case payload.Reason != "":
			ae.Message = payload.Reason
		case len(payload.Errors) > 0:
			// Some Bamboo endpoints (e.g. POST /queue/deployment validation)
			// put the human-readable text under `errors[]` without a top-level
			// `message`.
			ae.Message = strings.Join(payload.Errors, "; ")
		}
		ae.Errors = payload.Errors
	}
	if ae.Message == "" {
		// fall back to a truncated body
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 240 {
			snippet = snippet[:240] + "…"
		}
		ae.Message = snippet
	}
	if ae.Message == "" {
		ae.Message = http.StatusText(resp.StatusCode)
	}
	return ae.ToFail()
}
