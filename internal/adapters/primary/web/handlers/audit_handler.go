package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/yvan/nexora-core/internal/adapters/primary/web/middleware"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

type AuditHandler struct {
	auditService *services.AuditService
}

func NewAuditHandler(service *services.AuditService) *AuditHandler {
	return &AuditHandler{auditService: service}
}

// --- MINI-PARSERS (Pour effondrer la complexité cognitive S3776) ---

func parseAction(raw string) domain.ActionType {
	action := domain.ActionType(raw)
	switch action {
	case "LOGIN_SUCCESS", "LOGIN_FAILED", "LOGOUT", "NAS_UPDATE", "USER_CREATED":
		return action
	default:
		if raw != "" {
			log.Printf("⚠️ [AuditHandler] Action de filtre invalide ignorée: %s", raw)
		}
		return ""
	}
}

func parseUserIDParam(uIDStr string) *domain.UserID {
	if uIDStr == "" {
		return nil
	}
	uid, err := domain.ParseUserID(uIDStr)
	if err != nil {
		log.Printf("⚠️ [AuditHandler] UserID invalide ignoré: %s", uIDStr)
		return nil
	}
	return &uid
}

func parsePagination(limitStr, offsetStr string) (int, int) {
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000
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
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		log.Printf("⚠️ [AuditHandler] Format %s invalide ignoré: %s", fieldName, dateStr)
		return time.Time{}
	}
	return t
}

// buildAuditFilterFromRequest assemble le filtre proprement
func buildAuditFilterFromRequest(r *http.Request, tID domain.TenantID) domain.AuditFilter {
	query := r.URL.Query()
	limit, offset := parsePagination(query.Get("limit"), query.Get("offset"))

	return domain.AuditFilter{
		TenantID:  tID,
		IPAddress: query.Get("ip_address"),
		Action:    parseAction(query.Get("action")),
		UserID:    parseUserIDParam(query.Get("user_id")),
		Limit:     limit,
		Offset:    offset,
		StartDate: parseDate(query.Get("start_date"), "start_date"),
		EndDate:   parseDate(query.Get("end_date"), "end_date"),
	}
}

// -----------------------------------------------------------------------------

// GetLogs gère la route GET /v1/audit
func (h *AuditHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	// 1. SÉCURITÉ CRITIQUE : Récupération du TenantID depuis le JWT via le package middleware !
	tenantIDStr, ok := r.Context().Value(middleware.TenantIDKey).(string)
	if !ok || tenantIDStr == "" {
		log.Println("🚨 [Security] Tentative d'accès API Audit sans TenantID dans le context JWT")
		writeError(w, http.StatusUnauthorized, "Unauthorized: Invalid or missing Tenant ID in token")
		return
	}

	tID, err := domain.ParseTenantID(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Bad Request: Malformed Tenant ID in token")
		return
	}

	// 2. Construction du filtre (Complexité 0 grâce aux mini-parsers)
	filter := buildAuditFilterFromRequest(r, tID)

	// 3. Appel du Service
	logs, err := h.auditService.GetLogs(r.Context(), filter)
	if err != nil {
		log.Printf("❌ [AuditHandler] Erreur service GetLogs: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if logs == nil {
		logs = make([]domain.AuditLog, 0) // Évite de renvoyer "null" au frontend
	}

	// 4. Réponse JSON via ton super helper défini dans auth_handler.go
	writeJSON(w, http.StatusOK, SuccessResponse{
		Status: "success",
		Data: map[string]interface{}{
			"logs": logs,
			"meta": map[string]interface{}{
				"limit":  filter.Limit,
				"offset": filter.Offset,
				"count":  len(logs),
			},
		},
	})
}
