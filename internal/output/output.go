package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		if IsAgentMode() {
			return FormatJSON, nil
		}
		return FormatTable, nil
	case "table", "text":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unknown output format %q (want table|json|yaml)", s)
	}
}

// Renderable is implemented by values that know how to print themselves as a table.
// For json/yaml the value itself is marshalled directly.
type Renderable interface {
	RenderTable(w io.Writer) error
}

func Print(format Format, v any) error {
	return PrintTo(os.Stdout, format, v)
}

func PrintTo(w io.Writer, format Format, v any) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case FormatYAML:
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		defer enc.Close()
		return enc.Encode(v)
	case FormatTable:
		if r, ok := v.(Renderable); ok {
			return r.RenderTable(w)
		}
		// Fallback for primitives / unstructured maps: pretty-print as YAML.
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		defer enc.Close()
		return enc.Encode(v)
	}
	return fmt.Errorf("unsupported format: %s", format)
}

// PrintError emits a structured error to stderr in the requested format.
func PrintError(format Format, err error) {
	switch format {
	case FormatJSON, FormatYAML:
		payload := map[string]any{"error": map[string]any{"message": err.Error()}}
		if fe, ok := asFailError(err); ok {
			payload = map[string]any{"error": fe}
		}
		_ = PrintTo(os.Stderr, format, payload)
	default:
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
	}
}
