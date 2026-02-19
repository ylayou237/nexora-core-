package handlers

import (
	"encoding/json"
	"errors" // Ajouté pour errors.Is
	"net/http"

	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// LoginRequest définit le format JSON attendu
type LoginRequest struct {
	TenantID string `json:"tenant_id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// ErrorResponse structure uniforme pour les erreurs JSON
type ErrorResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Code   string `json:"code,omitempty"` // Optionnel : pour aider le frontend
}

// SuccessResponse structure uniforme pour succès JSON
type SuccessResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// Login gère la route POST /v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	tID, err := domain.NewTenantID(req.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Tenant ID")
		return
	}

	username, err := domain.NewUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Username")
		return
	}

	// 3️⃣ Appel du Service
	tokenPair, err := h.authService.Login(r.Context(), tID, username, req.Password)
	if err != nil {
		h.handleAuthError(w, err) // Extraction de la logique d'erreur
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{
		Status: "success",
		Data: map[string]interface{}{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_at":    tokenPair.ExpiresAt,
		},
	})
}

// handleAuthError transforme les erreurs du domaine en réponses HTTP appropriées
func (h *AuthHandler) handleAuthError(w http.ResponseWriter, err error) {
	// 🛡️ Cas spécifique : Compte bloqué (Brute Force)
	if errors.Is(err, domain.ErrAccountLocked) {
		writeError(w, http.StatusLocked, "Compte temporairement bloqué suite à trop d'échecs. Réessayez dans 15 minutes.")
		return
	}

	// ❌ Cas spécifique : Identifiants incorrects ou utilisateur inexistant
	if errors.Is(err, domain.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "Identifiants invalides")
		return
	}

	// ⏳ Cas spécifique : Compte expiré
	if errors.Is(err, domain.ErrUserExpired) {
		writeError(w, http.StatusForbidden, "Votre compte a expiré. Veuillez contacter le support.")
		return
	}

	// Par défaut
	writeError(w, http.StatusUnauthorized, err.Error())
}

// Helpers pour écrire JSON
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Status: "error",
		Error:  msg,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
