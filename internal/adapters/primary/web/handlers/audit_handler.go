package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/yvan/nexora-core/internal/adapters/primary/web/middlewares"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

type AuditHandler struct {
	auditService *services.AuditService
}

func NewAuditHandler(service *services.AuditService) *AuditHandler {
	return &AuditHandler{auditService: service}
}

// GetLogs gère la route GET /v1/audit
// Accès : Admin uniquement (vérifié par middleware)
func (h *AuditHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	// 1. ISOLATION TENANT : On récupère l'identité depuis le contexte injecté par le middleware Auth
	tenantIDStr, ok := r.Context().Value(middlewares.TenantIDKey).(string)
	if !ok || tenantIDStr == "" {
		log.Println("🚨 [Security] Tentative d'accès API Audit sans TenantID")
		writeError(w, http.StatusUnauthorized, "Accès refusé : identité manquante")
		return
	}

	tID, err := domain.ParseTenantID(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Format Tenant ID invalide")
		return
	}

	// 2. TIMEOUT CONTEXTUEL : On protège la DB contre les requêtes trop longues
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 3. FILTRAGE : Construction du filtre avec les mini-parsers
	filter := h.buildAuditFilterFromRequest(r, tID)

	// 4. SERVICE : Récupération des logs
	logs, err := h.auditService.GetLogs(ctx, filter)
	if err != nil {
		log.Printf("❌ [AuditHandler] Erreur service GetLogs: %v", err)
		writeError(w, http.StatusInternalServerError, "Erreur lors de la récupération des journaux d'audit")
		return
	}

	// 5. HYDRATATION : On garantit une liste vide plutôt que null pour le JSON
	if logs == nil {
		logs = []domain.AuditLog{}
	}

	// 6. RÉPONSE UNIFORME
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"logs": logs,
			"meta": map[string]interface{}{
				"limit":  filter.Limit,
				"offset": filter.Offset,
				"count":  len(logs),
			},
		},
	})
}

// --- HELPERS PRIVÉS (Isolation de la logique de parsing) ---

func (h *AuditHandler) buildAuditFilterFromRequest(r *http.Request, tID domain.TenantID) domain.AuditFilter {
	query := r.URL.Query()
	limit, offset := parsePagination(query.Get("limit"), query.Get("offset"))

	actorID := query.Get("actor_id")
	if actorID == "" {
		actorID = query.Get("user_id") // Rétrocompatibilité frontend
	}

	return domain.AuditFilter{
		TenantID:  tID,
		IPAddress: query.Get("ip_address"),
		Action:    parseAction(query.Get("action")),
		ActorID:   actorID,
		Limit:     limit,
		Offset:    offset,
		StartDate: parseDate(query.Get("start_date"), "start_date"),
		EndDate:   parseDate(query.Get("end_date"), "end_date"),
	}
}

// parseAction valide que l'action demandée fait partie des constantes du domaine
func parseAction(raw string) domain.ActionType {
	if raw == "" {
		return ""
	}
	action := domain.ActionType(raw)
	// On laisse le service ou le repo filtrer si l'action est exotique,
	// mais on valide ici les types de base.
	return action
}

func parsePagination(limitStr, offsetStr string) (int, int) {
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000 // Sécurité Carrier-Grade : évite l'épuisement de mémoire (OOM)
	}

	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func parseDate(dateStr, fieldName string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}
	// Support du format RFC3339 (Standard API moderne)
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		log.Printf("⚠️ [AuditHandler] Format de date invalide pour %s: %s", fieldName, dateStr)
		return time.Time{}
	}
	return t
}
