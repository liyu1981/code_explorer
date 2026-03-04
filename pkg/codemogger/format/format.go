package format

import (
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
)

func JSON(results []codemogger.SearchResult) ([]byte, error) {
	return json.MarshalIndent(results, "", "  ")
}

func Text(results []codemogger.SearchResult) string {
	var output string
	for _, r := range results {
		output += fmt.Sprintf("%s:%d - %s (%s)\n", r.FilePath, r.StartLine, r.Name, r.Kind)
		output += fmt.Sprintf("  Score: %.2f\n", r.Score)
		if r.Signature != "" {
			output += fmt.Sprintf("  %s\n", r.Signature)
		}
		if r.Snippet != "" {
			output += fmt.Sprintf("  ---\n  %s\n", r.Snippet)
		}
		output += "\n"
	}
	return output
}
