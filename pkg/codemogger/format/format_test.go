package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
)

func TestJSONFormat(t *testing.T) {
	results := []codemogger.SearchResult{
		{
			FilePath:  "main.go",
			StartLine: 1,
			Name:      "main",
			Kind:      "function",
			Score:     0.95,
		},
	}

	data, err := JSON(results)
	if err != nil {
		t.Fatalf("JSON formatting failed: %v", err)
	}

	var decoded []codemogger.SearchResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON decoding failed: %v", err)
	}

	if len(decoded) != 1 || decoded[0].Name != "main" {
		t.Error("JSON formatting produced incorrect data")
	}
}

func TestTextFormat(t *testing.T) {
	results := []codemogger.SearchResult{
		{
			FilePath:  "main.go",
			StartLine: 1,
			Name:      "main",
			Kind:      "function",
			Score:     0.95,
			Signature: "func main()",
		},
	}

	output := Text(results)
	if !strings.Contains(output, "main.go:1") {
		t.Errorf("Text format missing FilePath:Line, got:\n%s", output)
	}
	if !strings.Contains(output, "main (function)") {
		t.Errorf("Text format missing Name (Kind), got:\n%s", output)
	}
	if !strings.Contains(output, "Score: 0.95") {
		t.Errorf("Text format missing Score, got:\n%s", output)
	}
	if !strings.Contains(output, "func main()") {
		t.Errorf("Text format missing Signature, got:\n%s", output)
	}
}
