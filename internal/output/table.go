package output

import (
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
)

// NewTable returns a pre-configured table writer with rounded style and the writer attached.
func NewTable(w io.Writer, header ...any) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetStyle(table.StyleLight)
	if len(header) > 0 {
		t.AppendHeader(header)
	}
	return t
}
