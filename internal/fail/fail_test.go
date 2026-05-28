package fail

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrorString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		err   *Error
		want  string
	}{
		{
			name: "with http status",
			err:  &Error{Code: "not_found", Message: "missing", HTTPStatus: 404},
			want: "not_found: missing (http 404)",
		},
		{
			name: "without http status",
			err:  &Error{Code: "bad_input", Message: "nope"},
			want: "bad_input: nope",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	e := New("auth_error", "no token", ExitAuth)
	if e.Code != "auth_error" || e.Message != "no token" || e.Exit != ExitAuth {
		t.Fatalf("unexpected error: %+v", e)
	}
}

func TestWrap(t *testing.T) {
	t.Parallel()
	t.Run("nil returns nil", func(t *testing.T) {
		if got := Wrap(nil, "x", ExitGeneric); got != nil {
			t.Fatalf("Wrap(nil) = %v, want nil", got)
		}
	})
	t.Run("wraps message", func(t *testing.T) {
		base := errors.New("base reason")
		got := Wrap(base, "wrap_code", ExitUsage)
		if got == nil || got.Code != "wrap_code" || got.Message != "base reason" || got.Exit != ExitUsage {
			t.Fatalf("Wrap returned unexpected: %+v", got)
		}
	})
}

func TestExitCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil -> OK", nil, ExitOK},
		{"plain error -> generic", errors.New("boom"), ExitGeneric},
		{"fail error with exit", New("auth", "nope", ExitAuth), ExitAuth},
		{"fail error with zero exit -> generic", &Error{Code: "x", Message: "y"}, ExitGeneric},
		{"wrapped fail error", fmt.Errorf("ctx: %w", New("u", "v", ExitUsage)), ExitUsage},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ExitCode(tc.err); got != tc.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func TestErrorImplementsErrorInterface(t *testing.T) {
	t.Parallel()
	var _ error = (*Error)(nil)
	// errors.As round-trip
	wrapped := fmt.Errorf("outer: %w", New("c", "m", ExitUsage))
	var fe *Error
	if !errors.As(wrapped, &fe) {
		t.Fatalf("errors.As did not unwrap to *Error")
	}
	if !strings.Contains(wrapped.Error(), "outer:") {
		t.Fatalf("expected outer prefix, got %q", wrapped.Error())
	}
}
