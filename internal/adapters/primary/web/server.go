package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/yvan/nexora-core/internal/adapters/primary/web/handlers"
	"github.com/yvan/nexora-core/internal/adapters/primary/web/middlewares"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

const (
	MethodNotAllowedMsg = "Méthode non autorisée"
	DefaultTimeout      = 30 * time.Second
)

type Server struct {
	mux             *http.ServeMux
	authHandler     *handlers.AuthHandler
	auditHandler    *handlers.AuditHandler
	rateLimiter     *middlewares.RateLimiter
	authService     *services.AuthService
	auditMiddleware func(http.Handler) http.Handler
}

func NewServer(
	authHandler *handlers.AuthHandler,
	auditHandler *handlers.AuditHandler,
	rateLimiter *middlewares.RateLimiter,
	authService *services.AuthService,
	auditMiddleware func(http.Handler) http.Handler,
) *Server {
	s := &Server{
		mux:             http.NewServeMux(),
		authHandler:     authHandler,
		auditHandler:    auditHandler,
		rateLimiter:     rateLimiter,
		authService:     authService,
		auditMiddleware: auditMiddleware,
	}

	s.routes()
	return s
}

// ServeHTTP satisfait l'interface http.Handler.
// On injecte les headers de sécurité globaux et on lance la chaîne de middlewares.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Sécurité HTTP (Defense in Depth)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none';")

	// 2. Audit & Tracing -> Router
	s.auditMiddleware(s.mux).ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Définition des middlewares spécialisés
	requireAuth := middlewares.RequireAuth(s.authService)
	adminOnly := middlewares.RequireRole(domain.RoleSuperAdmin, domain.RoleProviderAdmin)

	// --- 1. AUTHENTIFICATION (Publique + Rate Limited) ---
	s.mux.Handle("/v1/auth/login", s.rateLimiter.Limit(s.onlyPost(s.authHandler.Login)))
	s.mux.Handle("/v1/auth/verify-mfa", s.rateLimiter.Limit(s.onlyPost(s.authHandler.VerifyMFA)))
	s.mux.Handle("/v1/auth/refresh", s.rateLimiter.Limit(s.onlyPost(s.authHandler.RefreshToken)))

	// Logout : nécessite un token valide
	s.mux.Handle("/v1/auth/logout", requireAuth(s.onlyPost(s.authHandler.Logout)))

	// --- 2. AUDIT (Sécurisée : Auth + RBAC) ---
	// La chaîne la plus complète : Limit -> Auth -> Admin -> Verb -> Handler
	auditChain := s.rateLimiter.Limit(
		requireAuth(
			adminOnly(
				s.onlyGet(s.auditHandler.GetLogs),
			),
		),
	)
	s.mux.Handle("/v1/audit", auditChain)

	// --- 3. SYSTÈME (Monitoring & K8s) ---
	s.mux.Handle("/health", s.onlyGet(s.handleHealth))
	s.mux.Handle("/ready", s.onlyGet(s.handleReady))
}

// --- DECORATORS DE ROUTAGE ---

func (s *Server) onlyPost(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			s.writeJSONError(w, MethodNotAllowedMsg, http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) onlyGet(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			s.writeJSONError(w, MethodNotAllowedMsg, http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- HANDLERS SYSTÈME ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]string{
		"status":  "ok",
		"service": "nexora-core",
		"time":    time.Now().Format(time.RFC3339),
	}, http.StatusOK)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Vérification réelle de la DB et de Redis via le service d'auth
	if err := s.authService.CheckIntegrity(ctx); err != nil {
		log.Printf("server: readiness probe failed: %v", err)
		s.writeJSONError(w, "Infrastructure Unhealthy", http.StatusServiceUnavailable)
		return
	}

	s.writeJSON(w, map[string]string{"status": "ready"}, http.StatusOK)
}

// --- HELPERS JSON ---

func (s *Server) writeJSON(w http.ResponseWriter, payload interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) writeJSONError(w http.ResponseWriter, message string, code int) {
	s.writeJSON(w, map[string]interface{}{
		"status": "error",
		"error":  message,
		"code":   code,
	}, code)
}
