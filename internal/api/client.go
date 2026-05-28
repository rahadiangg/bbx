package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rahadiangg/bbx/internal/version"
)

// Client is a minimal Bamboo Server REST client.
// It targets the `/rest/api/latest` namespace by default.
type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	userAgent  string
}

// Options configures a Client.
type Options struct {
	BaseURL            string
	Token              string
	InsecureSkipVerify bool
	Timeout            time.Duration
}

// New constructs a Client. BaseURL should be the Bamboo root, e.g. https://bamboo.example.com.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if opts.Token == "" {
		return nil, fmt.Errorf("token is required")
	}
	u, err := url.Parse(strings.TrimRight(opts.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if opts.InsecureSkipVerify {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // user opt-in
	}
	return &Client{
		baseURL:    u,
		token:      opts.Token,
		httpClient: &http.Client{Transport: tr, Timeout: timeout},
		userAgent:  fmt.Sprintf("bbx/%s", version.Version),
	}, nil
}

// BaseURL exposes the configured base URL (read-only).
func (c *Client) BaseURL() string { return c.baseURL.String() }

// Do executes a request against /rest/<path>. `apiPath` should start with `/api/latest/...`
// or `/...` for non-/rest paths (handled via path starting with `/rest/`).
func (c *Client) Do(ctx context.Context, method, apiPath string, query url.Values, body any, out any) error {
	full := c.resolve(apiPath)
	if len(query) > 0 {
		full.RawQuery = query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, full.String(), reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, full.Path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseError(resp)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		// drain to allow connection reuse
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// resolve joins an API path onto the base URL. If `p` already starts with `/rest`
// it is used as-is; otherwise it is prefixed with `/rest`.
func (c *Client) resolve(p string) *url.URL {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasPrefix(p, "/rest/") {
		p = "/rest" + p
	}
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + p
	return &u
}

// doRawJSON performs a request and returns the raw response body. Used by
// endpoints whose response shape varies across Bamboo versions and is best
// decoded by the caller (e.g. ClonePlan).
func (c *Client) doRawJSON(ctx context.Context, method, apiPath string, query url.Values, body any) ([]byte, error) {
	full := c.resolve(apiPath)
	if len(query) > 0 {
		full.RawQuery = query.Encode()
	}
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, full.String(), reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, full.Path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, parseError(resp)
	}
	return io.ReadAll(resp.Body)
}

// doDownload calls a path that lives OUTSIDE Bamboo's /rest namespace
// (e.g. /download/PROJ-A-JOB1/build_logs/PROJ-A-JOB1-1.log). The resolve()
// helper above always prefixes /rest, so we go through the raw base URL here.
//
// IMPORTANT: Bamboo Server 8.2.4 serves /download/* via a different servlet
// that requires *session-cookie* auth, not the Bearer PAT used by /rest/*.
// PAT requests get redirected to an HTML login page (with HTTP 200, content-type
// text/html). Callers detect this case by inspecting the returned content-type
// and surface a clean session_auth_required error.
func (c *Client) doDownload(ctx context.Context, path string) (contentType string, body []byte, err error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("GET %s: %w", u.Path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", nil, parseError(resp)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	return resp.Header.Get("Content-Type"), b, nil
}

// GetRaw fetches a path and returns the raw response body bytes. Used for the
// few endpoints that return text/plain or other non-JSON shapes.
func (c *Client) GetRaw(ctx context.Context, apiPath string, query url.Values) ([]byte, error) {
	full := c.resolve(apiPath)
	if len(query) > 0 {
		full.RawQuery = query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, parseError(resp)
	}
	return io.ReadAll(resp.Body)
}
