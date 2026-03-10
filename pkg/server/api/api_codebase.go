package api

import (
	"net/http"
)

func (h *ApiHandler) handleListSystemCodebases(w http.ResponseWriter, r *http.Request) {
	codebases, err := h.index.GetStore().ListCodebases()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list system codebases", err)
		return
	}
	writeJSON(w, http.StatusOK, codebases)
}
