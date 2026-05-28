package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/rahadiangg/bbx/internal/fail"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		in     string
		want   Format
		errStr string
	}{
		{"json", FormatJSON, ""},
		{"JSON", FormatJSON, ""},
		{" yaml ", FormatYAML, ""},
		{"yml", FormatYAML, ""},
		{"table", FormatTable, ""},
		{"text", FormatTable, ""},
		{"xml", "", "unknown output format"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseFormat(tc.in)
			if tc.errStr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errStr) {
					t.Fatalf("err = %v, want contains %q", err, tc.errStr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseFormatAutoUsesAgentMode(t *testing.T) {
	// Setting BBX_AGENT_MODE forces JSON in auto mode.
	t.Setenv("BBX_AGENT_MODE", "1")
	got, err := ParseFormat("")
	if err != nil {
		t.Fatal(err)
	}
	if got != FormatJSON {
		t.Fatalf("expected JSON in agent mode, got %s", got)
	}
}

type fakeRenderable struct{ called bool }

func (f *fakeRenderable) RenderTable(w io.Writer) error {
	f.called = true
	_, _ = io.WriteString(w, "RENDERED\n")
	return nil
}

func TestPrintTo(t *testing.T) {
	t.Run("json marshals indented", func(t *testing.T) {
		var buf bytes.Buffer
		if err := PrintTo(&buf, FormatJSON, map[string]int{"a": 1}); err != nil {
			t.Fatal(err)
		}
		var got map[string]int
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid json: %v\n%s", err, buf.String())
		}
		if got["a"] != 1 {
			t.Fatalf("unexpected payload: %v", got)
		}
		if !strings.Contains(buf.String(), "  ") {
			t.Fatalf("expected indentation in %q", buf.String())
		}
	})
	t.Run("yaml encodes", func(t *testing.T) {
		var buf bytes.Buffer
		if err := PrintTo(&buf, FormatYAML, map[string]string{"k": "v"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "k: v") {
			t.Fatalf("unexpected yaml: %q", buf.String())
		}
	})
	t.Run("table calls Renderable", func(t *testing.T) {
		var buf bytes.Buffer
		r := &fakeRenderable{}
		if err := PrintTo(&buf, FormatTable, r); err != nil {
			t.Fatal(err)
		}
		if !r.called {
			t.Fatal("RenderTable was not invoked")
		}
		if !strings.Contains(buf.String(), "RENDERED") {
			t.Fatalf("output missing render marker: %q", buf.String())
		}
	})
	t.Run("table falls back to yaml for primitives", func(t *testing.T) {
		var buf bytes.Buffer
		if err := PrintTo(&buf, FormatTable, map[string]string{"k": "v"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "k: v") {
			t.Fatalf("expected yaml fallback, got %q", buf.String())
		}
	})
	t.Run("unsupported format", func(t *testing.T) {
		if err := PrintTo(io.Discard, Format("nope"), nil); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestPrintErrorJSON(t *testing.T) {
	// Capture stderr by temporarily redirecting it.
	// PrintError writes to os.Stderr; we re-route via a pipe.
	r, w, err := newPipe()
	if err != nil {
		t.Fatal(err)
	}
	restore := redirectStderr(t, w)
	defer restore()

	PrintError(FormatJSON, fail.New("auth", "denied", fail.ExitAuth))
	_ = w.Close()
	got, _ := io.ReadAll(r)
	var payload map[string]any
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, got)
	}
	errMap, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("error payload missing: %v", payload)
	}
	if errMap["code"] != "auth" || errMap["message"] != "denied" {
		t.Fatalf("unexpected payload: %v", errMap)
	}
}

func TestPrintErrorPlainErrorJSON(t *testing.T) {
	r, w, _ := newPipe()
	restore := redirectStderr(t, w)
	defer restore()

	PrintError(FormatJSON, errors.New("boom"))
	_ = w.Close()
	got, _ := io.ReadAll(r)
	if !strings.Contains(string(got), "boom") {
		t.Fatalf("output missing boom: %q", got)
	}
}

func TestPrintErrorTable(t *testing.T) {
	r, w, _ := newPipe()
	restore := redirectStderr(t, w)
	defer restore()

	PrintError(FormatTable, errors.New("boom"))
	_ = w.Close()
	got, _ := io.ReadAll(r)
	if !strings.Contains(string(got), "Error:") || !strings.Contains(string(got), "boom") {
		t.Fatalf("expected human error, got %q", got)
	}
}

func TestIsAgentMode(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"BBX_AGENT_MODE wins", map[string]string{"BBX_AGENT_MODE": "1"}, true},
		{"CLAUDECODE wins", map[string]string{"CLAUDECODE": "1"}, true},
		{"CLAUDE_CODE wins", map[string]string{"CLAUDE_CODE": "1"}, true},
		{"ANTHROPIC_API_KEY wins", map[string]string{"ANTHROPIC_API_KEY": "k"}, true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for _, k := range []string{"BBX_AGENT_MODE", "CLAUDECODE", "CLAUDE_CODE", "ANTHROPIC_API_KEY"} {
				t.Setenv(k, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if got := IsAgentMode(); got != tc.want {
				t.Fatalf("IsAgentMode = %v, want %v", got, tc.want)
			}
		})
	}
}
