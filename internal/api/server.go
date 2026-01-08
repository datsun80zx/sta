package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/datsun80zx/sta.git/internal/analytics"
)

// Server represents the API server
type Server struct {
	db         *sql.DB
	handler    *Handler
	httpServer *http.Server
}

// NewServer creates a new API server
func NewServer(db *sql.DB) *Server {
	analyticsService := analytics.NewService(db)
	handler := NewHandler(analyticsService)

	return &Server{
		db:      db,
		handler: handler,
	}
}

// Start starts the API server on the given address
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/api/health", s.handler.HealthCheck)
	mux.HandleFunc("/api/business-units", s.handleBusinessUnits)
	mux.HandleFunc("/api/business-units/", s.handleBusinessUnit)
	mux.HandleFunc("/api/technicians/", s.handleTechnician)

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("API server listening on %s\n", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// handleBusinessUnits routes to GetBusinessUnits
func (s *Server) handleBusinessUnits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.handler.GetBusinessUnits(w, r)
}

// handleBusinessUnit routes to GetBusinessUnit
func (s *Server) handleBusinessUnit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.handler.GetBusinessUnit(w, r)
}

// handleTechnician routes to GetTechnician
func (s *Server) handleTechnician(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.handler.GetTechnician(w, r)
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow requests from localhost during development
		if origin != "" && isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAllowedOrigin checks if the origin is allowed for CORS
func isAllowedOrigin(origin string) bool {
	// Allow localhost for development
	if strings.HasPrefix(origin, "http://localhost") ||
		strings.HasPrefix(origin, "http://127.0.0.1") {
		return true
	}
	// Add production origins here as needed
	return false
}
