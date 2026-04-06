package server

import (
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/server/api"
	index "github.com/liyu1981/code_explorer/pkg/zoekt/index"
)

func New(idx *codemogger.CodeIndex, zIdx *index.ZoektIndex) http.Handler {
	apiHandler := api.NewHandler(&api.ApiConfig{
		CodemoggerIndex: idx,
		ZoektIndex:      zIdx,
	})

	uiServer := NewUIServer(&Config{
		ApiHandler: apiHandler,
	})

	return uiServer.SetupRoutes()
}
