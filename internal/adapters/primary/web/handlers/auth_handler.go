package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/ports"
	"github.com/yvan/nexora-core/internal/core/services"
)

// ======================= JSON DTO =======================

type LoginRequest struct {
	TenantID string `json:"tenant_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	DeviceID string `json:"device_id,omitempty"`
}

type MFARequest struct {
	TenantID string `json:"tenant_id"`
	Username string `json:"username"`
	Code     string `json:"code"`
	DeviceID string `json:"device_id,omitempty"`
}
type RefreshRequest struct {
	// 🚨 IMPORTANT : Le tag JSON doit être "refresh_token"
	// pour correspondre à ton script PowerShell !
	RefreshTokenRaw string `json:"refresh_token"`
}
type LogoutRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"` // 👈 Ajoute cette ligne pour la Révocation Nucléaire
	DeviceID     string `json:"device_id,omitempty"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
	MFARequired  bool   `json:"mfa_required,omitempty"`
	Message      string `json:"message,omitempty"`
}

// ======================= HANDLER DEFINITION =======================

type AuthHandler struct {
	authService *services.AuthService
	userRepo    ports.UserRepository
	timeout     time.Duration
}

func NewAuthHandler(authSvc *services.AuthService, userRepo ports.UserRepository, timeout time.Duration) *AuthHandler {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &AuthHandler{
		authService: authSvc,
		userRepo:    userRepo,
		timeout:     timeout,
	}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/login", h.Login)
	r.Post("/verify-mfa", h.VerifyMFA)
	r.Post("/refresh-token", h.RefreshToken)
	r.Post("/logout", h.Logout)
}

// ======================= HANDLERS =======================

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	var req LoginRequest
	if err := decodeJSONStrict(w, r, &req); err != nil {
		return
	}

	if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.Username) == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Champs requis manquants")
		return
	}

	ip, ua, traceID := extractSIEMContext(r)
	device := pickDeviceID(req.DeviceID, r, ip, ua)

	tokenPair, mfaRequired, err := h.authService.Login(ctx, services.LoginRequest{
		TenantID:  domain.TenantID(req.TenantID),
		Username:  domain.Username(req.Username),
		Password:  req.Password,
		IPAddr:    ip,
		UserAgent: ua,
		DeviceID:  device,
		TraceID:   traceID,
	})
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	if mfaRequired {
		writeJSON(w, http.StatusAccepted, TokenResponse{
			MFARequired: true,
			Message:     "Authentification MFA requise. Code envoyé.",
		})
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
	})
}

func (h *AuthHandler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	var req MFARequest
	if err := decodeJSONStrict(w, r, &req); err != nil {
		return
	}

	ip, ua, traceID := extractSIEMContext(r)
	device := pickDeviceID(req.DeviceID, r, ip, ua)

	user, err := h.userRepo.GetByUsername(ctx, domain.TenantID(req.TenantID), domain.Username(req.Username))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Utilisateur introuvable")
		return
	}

	tokenPair, err := h.authService.VerifyMFA(ctx, services.VerifyMFARequest{
		User:   user,
		Code:   req.Code,
		IP:     ip,
		UA:     ua,
		Device: device,
		Trace:  traceID,
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Code OTP incorrect ou expiré")
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
	})
}

// Dans internal/adapters/primary/web/handlers/auth_handler.go

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// 1. Timeout contextuel (Comme dans Login)
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	var req RefreshRequest

	// 2. Parseur Strict (Ferme la porte aux payloads malveillants)
	if err := decodeJSONStrict(w, r, &req); err != nil {
		return
	}

	if strings.TrimSpace(req.RefreshTokenRaw) == "" {
		writeError(w, http.StatusBadRequest, "Token de rafraîchissement manquant")
		return
	}

	// 🌟 3. EXTRACTION DU CONTEXTE (Vital pour l'Audit et le script Lua Redis)
	ip, ua, traceID := extractSIEMContext(r)
	device := pickDeviceID("", r, ip, ua)

	refreshReq := services.RefreshRequest{
		RefreshTokenRaw: req.RefreshTokenRaw,
		IP:              ip,
		UA:              ua,
		Device:          device,  // Lua saura enfin dans quelle famille ranger le token !
		Trace:           traceID, // Postgres aura enfin une trace propre !
	}

	// 4. Appel au service métier
	tokens, err := h.authService.Refresh(ctx, refreshReq)
	if err != nil {
		h.handleAuthError(w, err) // Utilise ton propre helper d'erreurs !
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	var req LogoutRequest
	// decodeJSONStrict rejettera la requête si le JSON est malformé
	// ou contient des champs inconnus.
	if err := decodeJSONStrict(w, r, &req); err != nil {
		return
	}

	ip, ua, traceID := extractSIEMContext(r)
	device := pickDeviceID(req.DeviceID, r, ip, ua)

	// Transfert vers le domaine (AuthService)
	err := h.authService.Logout(ctx, services.LogoutRequest{
		AccessToken:  req.AccessToken,
		RefreshToken: req.RefreshToken, // ✅ Transfert du token à détruire
		IPAddr:       ip,
		UserAgent:    ua,
		DeviceID:     device,
		TraceID:      traceID,
	})

	if err != nil {
		// On renvoie une erreur générique pour ne pas fuiter l'état interne
		writeError(w, http.StatusUnauthorized, "Échec de la déconnexion")
		return
	}

	// Succès total
	writeJSON(w, http.StatusOK, map[string]string{"message": "Déconnexion réussie"})
}

// ======================= HELPERS & SECURITY =======================

func (h *AuthHandler) handleAuthError(w http.ResponseWriter, err error) {
	switch {
	// ✅ Regroupement des erreurs d'authentification (401)
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrInvalidRefresh),
		strings.Contains(err.Error(), "REPLAY_DETECTED"),
		strings.Contains(err.Error(), "TOKEN_NOT_FOUND"):
		writeError(w, http.StatusUnauthorized, "Identifiants invalides ou session expirée")

	// ✅ Lockout -> 429 + Retry-After si dispo
	case errors.Is(err, domain.ErrAccountLocked):
		retryAfterSec := 0
		var rl *domain.RateLimitError
		if errors.As(err, &rl) && rl.RetryAfter > 0 {
			retryAfterSec = int(rl.RetryAfter.Round(time.Second).Seconds())
			if retryAfterSec < 1 {
				retryAfterSec = 1
			}
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSec))
		}

		// Recommandé : 429
		writeError(w, http.StatusTooManyRequests, "Trop de tentatives. Réessayez plus tard.")

	case errors.Is(err, domain.ErrUserExpired):
		writeError(w, http.StatusForbidden, "Compte expiré. Contactez le support.")

	default:
		log.Printf("❌ [AuthHandler] Error: %v", err)
		writeError(w, http.StatusInternalServerError, "Erreur interne")
	}
}

const maxJSONBody = 1 << 20

func decodeJSONStrict(w http.ResponseWriter, r *http.Request, out interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(out); err != nil {
		writeError(w, http.StatusBadRequest, "Corps JSON invalide")
		return err
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "Données JSON supplémentaires refusées")
		return errors.New("trailing json")
	}
	return nil
}

func extractSIEMContext(r *http.Request) (ip, ua, trace string) {
	ip = extractIP(r)
	ua = r.UserAgent()
	if ua == "" {
		ua = "unknown_client"
	}
	trace = strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if trace == "" {
		trace = uuid.New().String()
	}
	return
}

func pickDeviceID(reqDevice string, r *http.Request, ip string, ua string) string {
	if d := strings.TrimSpace(reqDevice); d != "" {
		return d
	}
	if d := strings.TrimSpace(r.Header.Get("X-Device-ID")); d != "" {
		return d
	}
	hash := sha256.Sum256([]byte(ip + "|" + ua))
	return "fp_" + hex.EncodeToString(hash[:8])
}

func extractIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}
