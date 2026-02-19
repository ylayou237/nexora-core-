package domain

import (
	"time"
)

// --- Erreurs de Domaine ---

// --- Value Objects & Types ---

// UsageDelta représente la consommation brute reçue d'un paquet RADIUS Interim-Update.
type UsageDelta struct {
	InputOctets  uint64
	OutputOctets uint64
	SessionTime  uint64 // En secondes, tel que reçu du NAS
}

// PolicySnapshot est une copie immuable des droits de l'utilisateur lors du login.
// Cela évite de recalculer les forfaits à chaque paquet réseau.
type PolicySnapshot struct {
	DataQuota   uint64        `json:"data_quota"`   // 0 = Illimité (en Octets)
	MaxUpload   uint64        `json:"max_upload"`   // bps
	MaxDownload uint64        `json:"max_download"` // bps
	TimeQuota   time.Duration `json:"time_quota"`   // 0 = Illimité
}

// --- Aggregate Root: ActiveSession ---
// Cet objet vit dans Redis. Il est optimisé pour des écritures ultra-rapides (Heartbeats).
type ActiveSession struct {
	// Identity & Routing (Exportés pour JSON/Redis)
	ID      SessionID `json:"id"`
	UserID  UserID    `json:"user_id"`
	NasIP   string    `json:"nas_ip"`
	MacAddr *MAC      `json:"mac"`

	// Configuration Immuable
	Policy PolicySnapshot `json:"policy"`

	// Compteurs Volatiles (État actuel de la session)
	InputOctets  uint64        `json:"input_octets"`
	OutputOctets uint64        `json:"output_octets"`
	SessionTime  time.Duration `json:"session_time"`

	// Gestion du Bail (Soft State / Zombie Protection)
	StartedAt     time.Time     `json:"started_at"`
	LastAliveAt   time.Time     `json:"last_alive_at"`
	LeaseDuration time.Duration `json:"lease_duration"`
}

// --- Factory ---

// NewActiveSession initialise une nouvelle session avec les compteurs à zéro.
func NewActiveSession(
	id SessionID,
	userID UserID,
	nasIP string,
	mac *MAC,
	policy PolicySnapshot,
	leaseBuffer time.Duration,
	clock Clock,
) (*ActiveSession, error) {
	now := clock.Now()
	return &ActiveSession{
		ID:            id,
		UserID:        userID,
		NasIP:         nasIP,
		MacAddr:       mac,
		Policy:        policy,
		InputOctets:   0,
		OutputOctets:  0,
		SessionTime:   0,
		StartedAt:     now,
		LastAliveAt:   now,
		LeaseDuration: leaseBuffer,
	}, nil
}

// --- Parameter Objects (Pour SonarQube & Lisibilité) ---

// SessionSnapshot contient toutes les données brutes pour reconstruire une session depuis Redis.
type SessionSnapshot struct {
	ID            SessionID
	UserID        UserID
	NasIP         string
	MacAddr       *MAC
	Policy        PolicySnapshot
	InputOctets   uint64
	OutputOctets  uint64
	SessionTime   time.Duration
	StartedAt     time.Time
	LastAliveAt   time.Time
	LeaseDuration time.Duration
}

// --- Rehydration ---

// RehydrateActiveSession permet de reconstruire l'objet depuis les données Redis (1 seul paramètre !).
func RehydrateActiveSession(data SessionSnapshot) *ActiveSession {
	return &ActiveSession{
		ID:            data.ID,
		UserID:        data.UserID,
		NasIP:         data.NasIP,
		MacAddr:       data.MacAddr,
		Policy:        data.Policy,
		InputOctets:   data.InputOctets,
		OutputOctets:  data.OutputOctets,
		SessionTime:   data.SessionTime,
		StartedAt:     data.StartedAt,
		LastAliveAt:   data.LastAliveAt,
		LeaseDuration: data.LeaseDuration,
	}
}

// --- Logique Métier (Enforcement) ---

// Heartbeat traite la consommation d'un intervalle et valide les quotas.
func (s *ActiveSession) Heartbeat(delta UsageDelta, clock Clock) error {
	now := clock.Now()

	// 1. PROTECTION ZOMBIE : Si le NAS a disparu trop longtemps, on rejette.
	if now.Sub(s.LastAliveAt) > s.LeaseDuration {
		return ErrSessionExpired
	}

	// 2. SÉCURITÉ : Gestion des reboots NAS ou des deltas vides.
	if delta.InputOctets == 0 && delta.OutputOctets == 0 && delta.SessionTime == 0 {
		s.LastAliveAt = now
		return nil
	}

	// 3. VÉRIFICATION QUOTA DATA
	if s.Policy.DataQuota > 0 {
		currentTotal := s.InputOctets + s.OutputOctets
		if (currentTotal + delta.InputOctets + delta.OutputOctets) > s.Policy.DataQuota {
			return ErrQuotaExceeded
		}
	}

	// 4. VÉRIFICATION QUOTA TEMPS
	if s.Policy.TimeQuota > 0 {
		newDuration := s.SessionTime + (time.Duration(delta.SessionTime) * time.Second)
		if newDuration > s.Policy.TimeQuota {
			return ErrQuotaExceeded
		}
	}

	// 5. COMMIT : Mise à jour de l'état si tout est valide
	s.InputOctets += delta.InputOctets
	s.OutputOctets += delta.OutputOctets
	s.SessionTime += time.Duration(delta.SessionTime) * time.Second
	s.LastAliveAt = now

	return nil
}

// --- Helpers & Getters ---

func (s *ActiveSession) IsAlive(clock Clock) bool {
	return clock.Now().Sub(s.LastAliveAt) <= s.LeaseDuration
}

func (s *ActiveSession) TotalUsage() uint64 {
	return s.InputOctets + s.OutputOctets
}

// MAC retourne une copie profonde pour éviter toute mutation externe du pointeur.
func (s *ActiveSession) GetMAC() *MAC {
	if s.MacAddr == nil {
		return nil
	}
	clone := *s.MacAddr
	return &clone
}

// Getters pour assurer la compatibilité avec le reste du code
func (s *ActiveSession) GetID() SessionID        { return s.ID }
func (s *ActiveSession) GetUserID() UserID       { return s.UserID }
func (s *ActiveSession) GetNasIP() string        { return s.NasIP }
func (s *ActiveSession) GetStartedAt() time.Time { return s.StartedAt }
