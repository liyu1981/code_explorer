package server

import (
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/server/api"
)

func New(idx *codemogger.CodeIndex) http.Handler {
	apiHandler := api.NewHandler(&api.ApiConfig{
		CodemoggerIndex: idx,
	})

	uiServer := NewUIServer(&Config{
		ApiHandler: apiHandler,
	})

	return uiServer.SetupRoutes()
}
