package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/ports"
)

// Définition des clés de contexte pour éviter les collisions
type contextKey string

const TraceIDKey contextKey = "trace_id"
const ActorIDKey contextKey = "actor_id"

// responseWriter est un wrapper (décorateur) pour capturer le code de statut HTTP
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader intercepte le code HTTP (ex: 200, 401, 404) avant qu'il ne parte au client
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// AuditMiddleware est l'intercepteur global pour le SIEM
func AuditMiddleware(auditRepo ports.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			// 1. Récupération ou Génération du TraceID
			traceID := r.Header.Get("X-Trace-Id")
			if traceID == "" {
				traceID = uuid.New().String()
			}

			// 2. Injection du TraceID dans le contexte pour les services (ex: AuthService)
			ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
			r = r.WithContext(ctx)

			// 3. Wrapper le ResponseWriter pour capturer le statut final
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			// ==========================================
			// 🚀 EXÉCUTION DE LA REQUÊTE (Next Handler)
			// ==========================================
			next.ServeHTTP(rw, r)

			// ==========================================
			// 🛡️ POST-TRAITEMENT (Log d'Audit Asynchrone)
			// ==========================================

			// Tentative de récupération des infos d'identité si un middleware Auth est passé avant
			tenantStr, _ := r.Context().Value(TenantIDKey).(string)
			tenantID, _ := domain.ParseTenantID(tenantStr)

			actorID, _ := r.Context().Value(ActorIDKey).(string)
			if actorID == "" {
				actorID = "anonymous"
			}

			// Détermination du statut de l'audit en fonction du code HTTP
			status := domain.AuditStatusSuccess
			if rw.status >= 400 && rw.status < 500 {
				status = domain.AuditStatusWarning // Erreurs client (ex: 401 Unauthorized, 403 Forbidden)
			} else if rw.status >= 500 {
				status = domain.AuditStatusFailure // Erreurs serveur (ex: 500 Crash)
			}

			// Création de l'entrée d'audit
			logEntry := &domain.AuditLog{
				TraceID:   traceID,
				TenantID:  tenantID,
				ActorID:   actorID,
				ActorType: domain.ActorUser,
				Action:    "API_ACCESS", // Action générique de base
				Status:    status,
				IPAddress: r.RemoteAddr,
				UserAgent: r.UserAgent(),
				Metadata: map[string]interface{}{
					"method":      r.Method,
					"path":        r.URL.Path,
					"status_code": rw.status,
					"latency_ms":  time.Since(startTime).Milliseconds(),
				},
				CreatedAt: time.Now(),
			}

			// ⚠️ DÉTAIL CARRIER-GRADE : On utilise context.Background() !
			// Si on utilise r.Context(), l'insertion en base sera annulée dès que
			// la réponse HTTP sera envoyée au client (Context Canceled).
			go func(entry *domain.AuditLog) {
				_ = auditRepo.LogEvent(context.Background(), entry)
			}(logEntry)
		})
	}
}
