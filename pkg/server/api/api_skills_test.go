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

	// 1. List Skills (should have built-ins seeded by NewHandler)
	t.Run("ListSkills", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/skills", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var skills []db.Skill
		if err := json.NewDecoder(w.Body).Decode(&skills); err != nil {
			t.Fatal(err)
		}

		if len(skills) == 0 {
			t.Error("expected seeded skills, got none")
		}
	})

	// 2. Get Skill
	t.Run("GetSkill", func(t *testing.T) {
		// Get first skill name
		reqList := httptest.NewRequest("GET", "/api/skills", nil)
		wList := httptest.NewRecorder()
		mux.ServeHTTP(wList, reqList)
		var skills []db.Skill
		json.NewDecoder(wList.Body).Decode(&skills)
		skillName := skills[0].Name

		req := httptest.NewRequest("GET", "/api/skills/get?name="+skillName, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var skill db.Skill
		if err := json.NewDecoder(w.Body).Decode(&skill); err != nil {
			t.Fatal(err)
		}
		if skill.Name != skillName {
			t.Errorf("expected skill name %s, got %s", skillName, skill.Name)
		}
	})

	// 3. Update Skill
	t.Run("UpdateSkill", func(t *testing.T) {
		reqList := httptest.NewRequest("GET", "/api/skills", nil)
		wList := httptest.NewRecorder()
		mux.ServeHTTP(wList, reqList)
		var skills []db.Skill
		json.NewDecoder(wList.Body).Decode(&skills)
		skill := skills[0]

		skill.SystemPrompt = "Updated prompt"
		body, _ := json.Marshal(skill)
		req := httptest.NewRequest("PUT", "/api/skills", strings.NewReader(string(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify update
		updated, _ := index.GetStore().GetSkillByName(context.Background(), skill.Name)
		if updated.SystemPrompt != "Updated prompt" {
			t.Errorf("expected updated prompt, got %s", updated.SystemPrompt)
		}
	})

	// 4. Reset Skill
	t.Run("ResetSkill", func(t *testing.T) {
		reqList := httptest.NewRequest("GET", "/api/skills", nil)
		wList := httptest.NewRecorder()
		mux.ServeHTTP(wList, reqList)
		var skills []db.Skill
		json.NewDecoder(wList.Body).Decode(&skills)
		skillName := skills[0].Name

		req := httptest.NewRequest("POST", "/api/skills/reset?name="+skillName, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify reset (should not be "Updated prompt")
		reset, _ := index.GetStore().GetSkillByName(context.Background(), skillName)
		if reset.SystemPrompt == "Updated prompt" {
			t.Error("expected prompt to be reset, but it's still 'Updated prompt'")
		}
	})
}
