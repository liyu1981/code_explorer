package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func TestApiHandler_Skills(t *testing.T) {
	h, index, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// 1. List Agent Prompts (should have built-ins seeded by NewHandler)
	t.Run("ListAgentPrompts", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/agent_prompts", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var prompts []db.Prompt
		if err := json.NewDecoder(w.Body).Decode(&prompts); err != nil {
			t.Fatal(err)
		}

		if len(prompts) == 0 {
			t.Error("expected seeded prompts, got none")
		}
	})

	// 2. Get Prompt
	t.Run("GetPrompt", func(t *testing.T) {
		// Get first prompt name
		reqList := httptest.NewRequest("GET", "/api/agent_prompts", nil)
		wList := httptest.NewRecorder()
		mux.ServeHTTP(wList, reqList)
		var prompts []db.Prompt
		json.NewDecoder(wList.Body).Decode(&prompts)
		promptName := prompts[0].Name

		req := httptest.NewRequest("GET", "/api/agent_prompts/get?name="+promptName, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var prompt db.Prompt
		if err := json.NewDecoder(w.Body).Decode(&prompt); err != nil {
			t.Fatal(err)
		}
		if prompt.Name != promptName {
			t.Errorf("expected prompt name %s, got %s", promptName, prompt.Name)
		}
	})

	// 3. Update Prompt
	t.Run("UpdatePrompt", func(t *testing.T) {
		reqList := httptest.NewRequest("GET", "/api/agent_prompts", nil)
		wList := httptest.NewRecorder()
		mux.ServeHTTP(wList, reqList)
		var prompts []db.Prompt
		json.NewDecoder(wList.Body).Decode(&prompts)
		prompt := prompts[0]

		prompt.SystemPrompt = "Updated prompt"
		body, _ := json.Marshal(prompt)
		req := httptest.NewRequest("PUT", "/api/agent_prompts", strings.NewReader(string(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify update
		updated, _ := index.GetStore().GetPromptByName(context.Background(), prompt.Name)
		if updated.SystemPrompt != "Updated prompt" {
			t.Errorf("expected updated prompt, got %s", updated.SystemPrompt)
		}
	})
}
