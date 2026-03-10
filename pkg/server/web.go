package server

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/liyu1981/code_explorer/pkg/server/api"
	"github.com/liyu1981/code_explorer/pkg/util"
)

//go:embed all:ui/out
var frontendFS embed.FS

// UIServer represents the UI server
type UIServer struct {
	listenAddr string
	server     *http.Server
	staticFS   fs.FS
	ApiHandler *api.ApiHandler
}

// Config holds the UI server configuration
type Config struct {
	ListenAddr string
	ApiHandler *api.ApiHandler
}

// NewUIServer creates a new UI server instance
func NewUIServer(config *Config) *UIServer {
	staticFS, err := fs.Sub(frontendFS, "ui/out")
	if err != nil {
		log.Fatal(err)
	}

	return &UIServer{
		listenAddr: config.ListenAddr,
		staticFS:   staticFS,
		ApiHandler: config.ApiHandler,
	}
}

// SetupRoutes configures all UI routes
func (s *UIServer) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Install API routes FIRST (so they take precedence)
	s.ApiHandler.RegisterRoutes(mux)

	// Create custom file server handler for SPA routing
	fileServer := http.FileServer(http.FS(s.staticFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 1. If requesting root, just serve it (will find index.html)
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// 2. Remove trailing slash for HTML file lookup
		cleanPath := strings.TrimSuffix(path, "/")

		// 3. Check if the file exists exactly as requested
		_, err := s.staticFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// 4. If not found, try appending ".html" to the clean path
		htmlPath := strings.TrimPrefix(cleanPath, "/") + ".html"
		if _, err := s.staticFS.Open(htmlPath); err == nil {
			// Create a new request with modified path
			newReq := r.Clone(r.Context())
			newReq.URL.Path = cleanPath + ".html"
			fileServer.ServeHTTP(w, newReq)
			return
		}

		// 5. Fallback: Serve the standard file server (handles 404s)
		fileServer.ServeHTTP(w, r)
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
		// Don't log static assets to reduce noise
		if !strings.HasPrefix(r.URL.Path, "/_next") && !strings.HasPrefix(r.URL.Path, "/static") {
			log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
		}
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always enable CORS headers to support access via IP or different hostnames
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if util.IsDev() {
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
