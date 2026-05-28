package output

import (
	"os"
	"testing"
)

// redirectStderr swaps os.Stderr for the given writer; the returned function
// restores the original.
func redirectStderr(t *testing.T, w *os.File) func() {
	t.Helper()
	orig := os.Stderr
	os.Stderr = w
	return func() { os.Stderr = orig }
}

// newPipe wraps os.Pipe for ergonomics in tests.
func newPipe() (*os.File, *os.File, error) {
	return os.Pipe()
}
