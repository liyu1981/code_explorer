package api

import (
	"encoding/json"
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/prompt"
)

func (h *ApiHandler) handleListSkills(w http.ResponseWriter, r *http.Request) {
	skills, err := h.index.GetStore().ListAgentPrompts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list prompts", err)
		return
	}

	builtinNames, err := prompt.GetBuiltinPromptNames()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get builtin prompts", err)
		return
	}

	for i := range skills {
		if builtinNames[skills[i].Name] {
			skills[i].IsBuiltin = true
		}
	}

	writeJSON(w, http.StatusOK, skills)
}

func (h *ApiHandler) handleGetSkill(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required", nil)
		return
	}

	skill, err := h.index.GetStore().GetPromptByName(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get prompt", err)
		return
	}
	if skill == nil {
		writeError(w, http.StatusNotFound, "prompt not found", nil)
		return
	}

	builtinNames, err := prompt.GetBuiltinPromptNames()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get builtin prompts", err)
		return
	}

	if builtinNames[skill.Name] {
		skill.IsBuiltin = true
	}

	writeJSON(w, http.StatusOK, skill)
}

func (h *ApiHandler) handleUpdateSkill(w http.ResponseWriter, r *http.Request) {
	var skill db.Prompt
	if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if skill.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required", nil)
		return
	}

	if err := h.index.GetStore().UpdatePrompt(r.Context(), &skill); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update prompt", err)
		return
	}

	writeJSON(w, http.StatusOK, skill)
}

func (h *ApiHandler) handleDeleteSkill(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required", nil)
		return
	}

	skill, err := h.index.GetStore().GetPromptByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get prompt", err)
		return
	}
	if skill == nil {
		writeError(w, http.StatusNotFound, "prompt not found", nil)
		return
	}

	builtinNames, err := prompt.GetBuiltinPromptNames()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get builtin prompts", err)
		return
	}

	if builtinNames[skill.Name] {
		writeError(w, http.StatusForbidden, "cannot delete built-in prompt", nil)
		return
	}

	if err := h.index.GetStore().DeletePrompt(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete prompt", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "prompt deleted"})
}
