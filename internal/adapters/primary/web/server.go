package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/yvan/nexora-core/internal/adapters/primary/web/handlers"
	"github.com/yvan/nexora-core/internal/adapters/primary/web/middleware"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

const (
	MethodNotAllowedMsg = "Method Not Allowed"
)

// Server représente le serveur HTTP de l'API REST
type Server struct {
	mux          *http.ServeMux
	authHandler  *handlers.AuthHandler
	auditHandler *handlers.AuditHandler
	rateLimiter  *middleware.RateLimiter
	authService  *services.AuthService
}

// NewServer initialise le routeur et enregistre toutes les routes
func NewServer(
	authHandler *handlers.AuthHandler,
	auditHandler *handlers.AuditHandler,
	rateLimiter *middleware.RateLimiter,
	authService *services.AuthService,
) *Server {
	s := &Server{
		mux:          http.NewServeMux(),
		authHandler:  authHandler,
		auditHandler: auditHandler,
		rateLimiter:  rateLimiter,
		authService:  authService,
	}

	s.routes()
	return s
}

// ServeHTTP permet à Server de satisfaire l'interface http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// routes définit toutes les routes et middlewares
func (s *Server) routes() {
	// Middleware Auth
	requireAuth := middleware.RequireAuth(s.authService)

	// Middleware RBAC
	requireAdminOrProvider := middleware.RequireRole(domain.RoleSuperAdmin, domain.RoleProviderAdmin)

	// --- Routes publiques ---
	s.mux.Handle("/v1/auth/login", s.rateLimiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, MethodNotAllowedMsg, http.StatusMethodNotAllowed)
			return
		}
		s.authHandler.Login(w, r)
	})))

	// --- Routes sécurisées ---
	s.mux.Handle("/v1/audit",
		s.rateLimiter.Limit(
			requireAuth(
				requireAdminOrProvider(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodGet {
							writeJSONError(w, MethodNotAllowedMsg, http.StatusMethodNotAllowed)
							return
						}
						s.auditHandler.GetLogs(w, r)
					}),
				),
			),
		),
	)

	// --- Monitoring ---
	s.mux.HandleFunc("/health", s.wrapHealthHandler(s.handleHealth))
	s.mux.HandleFunc("/ready", s.wrapHealthHandler(s.handleReady))
}

// wrapHealthHandler ajoute la vérification du verbe HTTP pour Health / Ready
func (s *Server) wrapHealthHandler(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, MethodNotAllowedMsg, http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}

// handleHealth retourne le status OK
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"status":  "ok",
		"service": "nexora-core",
	}, http.StatusOK)
}

// handleReady retourne le status Ready
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Ici tu peux ajouter un PING Postgres / Redis
	writeJSON(w, map[string]string{
		"status": "ready",
	}, http.StatusOK)
}

// writeJSON simplifie l'écriture JSON standardisée
func writeJSON(w http.ResponseWriter, payload interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Erreur JSON response: %v", err)
	}
}

// writeJSONError pour uniformiser les erreurs HTTP en JSON
func writeJSONError(w http.ResponseWriter, message string, code int) {
	writeJSON(w, map[string]interface{}{
		"status": "error",
		"error":  message,
	}, code)
}
