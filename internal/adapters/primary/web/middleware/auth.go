package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

const (
	// Key unique pour stocker les claims dans le contexte
	AuthClaimsKey ContextKey = "auth_claims"
)

// RequireAuth middleware Carrier-Grade
// Valide le JWT via AuthService et injecte les claims dans le context.
func RequireAuth(authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized: Missing or invalid token", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Validation JWT via AuthService
			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "Unauthorized: Token expired or invalid", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), AuthClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAuthClaims récupère les claims depuis le contexte d'une requête
func GetAuthClaims(ctx context.Context) (*domain.UserClaims, bool) {
	claims, ok := ctx.Value(AuthClaimsKey).(*domain.UserClaims)
	return claims, ok
}

// RequireTenant middleware pour vérifier l'accès multi-tenant
func RequireTenant(tenantID domain.TenantID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetAuthClaims(r.Context())
			if !ok {
				http.Error(w, "Unauthorized: missing claims", http.StatusUnauthorized)
				return
			}

			if claims.TenantID != tenantID {
				http.Error(w, "Forbidden: tenant mismatch", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
