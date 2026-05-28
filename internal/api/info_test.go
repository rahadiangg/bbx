package api

import (
	"context"
	"testing"
)

func TestGetServerInfo(t *testing.T) {
	t.Parallel()
	fb := newFakeBamboo(t)
	fb.expect("GET", "/rest/api/latest/info", 200,
		`{"version":"8.2.4","edition":"","buildDate":"2022-06-13T19:58:59.000+08:00","buildNumber":"80210","state":"RUNNING"}`)
	c := newTestClient(t, fb)

	info, err := c.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServerInfo: %v", err)
	}
	if info.Version != "8.2.4" || info.BuildNumber != "80210" || info.State != "RUNNING" {
		t.Errorf("info = %+v", info)
	}
}

func TestServerInfoMajorVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want int
	}{
		{"8.2.4", 8},
		{"10.0", 10},
		{"12.1.5", 12},
		{"", 0},
		{"garbage", 0},
		{"9", 9},
	}
	for _, tc := range tests {
		s := &ServerInfo{Version: tc.in}
		if got := s.MajorVersion(); got != tc.want {
			t.Errorf("MajorVersion(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
	// nil safety
	var nilInfo *ServerInfo
	if got := nilInfo.MajorVersion(); got != 0 {
		t.Errorf("nil.MajorVersion() = %d, want 0", got)
	}
}
