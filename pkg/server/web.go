package server

import (
	"log"
	"net/http"
	"time"

	"github.com/liyu1981/code_explorer/pkg/server/api"
	"github.com/liyu1981/code_explorer/pkg/util"
)

// UIServer represents the UI server
type UIServer struct {
	listenAddr string
	server     *http.Server
	ApiHandler *api.ApiHandler
}

// Config holds the UI server configuration
type Config struct {
	ListenAddr string
	ApiHandler *api.ApiHandler
}

// NewUIServer creates a new UI server instance
func NewUIServer(config *Config) *UIServer {
	return &UIServer{
		listenAddr: config.ListenAddr,
		ApiHandler: config.ApiHandler,
	}
}

// SetupRoutes configures all UI routes
func (s *UIServer) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Install API routes FIRST (so they take precedence)
	s.ApiHandler.RegisterRoutes(mux)

	// Fallback for root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("code_explorer server is running"))
			return
		}
		http.NotFound(w, r)
	})

	// Wrap with middleware
	return corsMiddleware(loggingMiddleware(mux))
}

// Start starts the UI server
func (s *UIServer) Start() error {
	s.server = &http.Server{
		Addr:         s.listenAddr,
		Handler:      s.SetupRoutes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("UI server listening on %s", s.listenAddr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the UI server
func (s *UIServer) Shutdown() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// Middleware

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// allow CORS when in dev mode
		if util.IsDev() {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
