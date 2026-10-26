package cli

import (
	"encoding/json"
	"io"
	"os"
)

// printJSON writes v to stdout as a single JSON document with a trailing
// newline and no decoration, so `--json` output is directly consumable by
// jq, scripts, and the Claude Code plugin.
func printJSON(v any) error {
	return printJSONTo(os.Stdout, v)
}

// printJSONTo is printJSON for a command's own writer.
func printJSONTo(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
