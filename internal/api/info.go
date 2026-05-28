package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

// ServerInfo describes the Bamboo instance bbx is talking to. It maps directly
// to GET /rest/api/latest/info, which is a stable, unauthenticated-readable
// endpoint going back to at least Bamboo 6.x — making it the best signal for
// "what version is on the other end of the wire."
type ServerInfo struct {
	Version     string `json:"version"`             // e.g. "8.2.4" or "9.6.1"
	Edition     string `json:"edition,omitempty"`   // empty on Server; "DC" on Data Center
	BuildNumber string `json:"buildNumber,omitempty"`
	BuildDate   string `json:"buildDate,omitempty"`
	State       string `json:"state,omitempty"`     // typically "RUNNING"
}

// MajorVersion returns the integer major-version component of Version, or 0
// if it cannot be parsed. Convenient for "if version.Major >= 10" gates.
func (s *ServerInfo) MajorVersion() int {
	if s == nil || s.Version == "" {
		return 0
	}
	parts := strings.SplitN(s.Version, ".", 2)
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return n
}

// GetServerInfo fetches version + edition info from the Bamboo instance.
// Cheap (one GET, no expansion). Safe to call eagerly during `auth login`.
func (c *Client) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	var info ServerInfo
	if err := c.Do(ctx, http.MethodGet, "/api/latest/info", nil, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
