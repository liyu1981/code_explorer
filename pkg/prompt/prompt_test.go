package prompt

import (
	"strings"
	"testing"
)

func TestGetPrompt(t *testing.T) {
	// We know research_instruction.md exists because it's in the repo
	content, err := GetPrompt("research_instruction.md")
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if len(content) == 0 {
		t.Errorf("expected non-empty content")
	}

	if !strings.Contains(content, "# Research Instruction") {
		t.Errorf("content does not seem to be research_instruction.md")
	}

	_, err = GetPrompt("non_existent.md")
	if err == nil {
		t.Errorf("expected error for non-existent prompt")
	}
}

func TestAllPrompts(t *testing.T) {
	prompts, err := AllPrompts()
	if err != nil {
		t.Fatalf("AllPrompts failed: %v", err)
	}

	found := false
	for _, p := range prompts {
		if p == "research_instruction.md" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("research_instruction.md not found in AllPrompts")
	}
}
