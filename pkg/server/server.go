package server

import (
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/server/api"
	"github.com/liyu1981/code_explorer/pkg/zoekt"
)

func New(cmIndex *codemogger.CodeIndex, zkIndex *zoekt.ZoektIndex) http.Handler {
	apiHandler := api.NewHandler(&api.ApiConfig{
		CodemoggerIndex: cmIndex,
		ZoektIndex:      zkIndex,
	})

	uiServer := NewUIServer(&Config{
		ApiHandler: apiHandler,
	})

	return uiServer.SetupRoutes()
}
