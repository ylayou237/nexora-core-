package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/yvan/nexora-core/internal/core/ports"
)

// ======================= CONFIG =======================

type JWKSConfig struct {
	KeySize         int
	RotationTTL     time.Duration // ex: 24h
	SyncInterval    time.Duration // ex: 1m
	LockTTL         time.Duration // ex: 10s (doit couvrir rotation+save)
	RetentionPeriod time.Duration // ex: 7j (>= max TTL refresh token idéalement)
	MaxKeys         int           // garde-fou anti croissance
	StateKeyLock    string        // ex: "jwks:lock:rotation"
}

func DefaultJWKSConfig() JWKSConfig {
	return JWKSConfig{
		KeySize:         2048,
		RotationTTL:     24 * time.Hour,
		SyncInterval:    1 * time.Minute,
		LockTTL:         10 * time.Second,
		RetentionPeriod: 7 * 24 * time.Hour,
		MaxKeys:         20,
		StateKeyLock:    "jwks:lock:rotation",
	}
}

// ======================= RFC 7517 STRUCTURES =======================

type JWK struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

// ======================= INTERFACE (AuthService depends on this) =======================

type JWKSService interface {
	GetCurrentPrivateKey() (*rsa.PrivateKey, string)
	GetPublicKeyByKid(kid string) (*rsa.PublicKey, error)
	GetJWKS() JWKS
	Stop()
}

// ======================= SERVICE =======================

type DistributedJWKSService struct {
	mu sync.RWMutex

	// Cache RAM (hot path)
	current    *rsa.PrivateKey
	currentKid string
	publicKeys map[string]*rsa.PublicKey
	createdAt  map[string]time.Time

	cfg   JWKSConfig
	store ports.JWKSStore
	log   *log.Logger
	stop  chan struct{}
}

func NewDistributedJWKSService(cfg JWKSConfig, store ports.JWKSStore, logger *log.Logger) (*DistributedJWKSService, error) {
	if store == nil {
		return nil, errors.New("jwks store is nil")
	}
	if logger == nil {
		logger = log.Default()
	}

	// garde-fous
	if cfg.KeySize < 2048 {
		cfg.KeySize = 2048
	}
	if cfg.RotationTTL <= 0 {
		cfg.RotationTTL = 24 * time.Hour
	}
	if cfg.SyncInterval <= 0 {
		cfg.SyncInterval = 1 * time.Minute
	}
	if cfg.LockTTL <= 0 {
		cfg.LockTTL = 10 * time.Second
	}
	if cfg.RetentionPeriod <= 0 {
		cfg.RetentionPeriod = 7 * 24 * time.Hour
	}
	if cfg.MaxKeys <= 0 {
		cfg.MaxKeys = 20
	}
	if cfg.StateKeyLock == "" {
		cfg.StateKeyLock = "jwks:lock:rotation"
	}

	s := &DistributedJWKSService{
		cfg:        cfg,
		store:      store,
		log:        logger,
		stop:       make(chan struct{}),
		publicKeys: map[string]*rsa.PublicKey{},
		createdAt:  map[string]time.Time{},
	}

	// Initialisation "fail-fast": on charge Redis, sinon on tente de devenir leader et créer la première clé.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.bootstrap(ctx); err != nil {
		return nil, err
	}

	go s.worker()
	return s, nil
}

func (s *DistributedJWKSService) Stop() {
	select {
	case <-s.stop:
		return
	default:
		close(s.stop)
	}
}

// ======================= PUBLIC API =======================

func (s *DistributedJWKSService) GetCurrentPrivateKey() (*rsa.PrivateKey, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current, s.currentKid
}

func (s *DistributedJWKSService) GetPublicKeyByKid(kid string) (*rsa.PublicKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pub, ok := s.publicKeys[kid]
	if !ok {
		return nil, errors.New("kid unknown")
	}
	return pub, nil
}

// Exposition JWKS (public keys only)
// Note: Pour encoder N/E en base64url, idéalement tu réutilises ta fonction existante.
// Ici, on reste simple: ton handler peut construire JWKS lui-même si tu veux.
// (Je te laisse la version "service" prête aussi via helper plus bas.)
func (s *DistributedJWKSService) GetJWKS() JWKS {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]JWK, 0, len(s.publicKeys))
	for kid, pub := range s.publicKeys {
		n, e := encodeRSAKeyParams(pub)
		keys = append(keys, JWK{
			Kty: "RSA",
			Alg: "RS256",
			Use: "sig",
			Kid: kid,
			N:   n,
			E:   e,
		})
	}

	// stabilité de sortie
	sort.Slice(keys, func(i, j int) bool { return keys[i].Kid < keys[j].Kid })
	return JWKS{Keys: keys}
}

// ======================= WORKER LOOP =======================

func (s *DistributedJWKSService) worker() {
	t := time.NewTicker(s.cfg.SyncInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = s.syncFromStore(ctx)
			cancel()

			// Rotation si nécessaire (lock distribué)
			if s.needsRotation() {
				ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
				s.attemptRotation(ctx2)
				cancel2()
			}

		case <-s.stop:
			return
		}
	}
}

func (s *DistributedJWKSService) needsRotation() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.current == nil || s.currentKid == "" {
		return true
	}
	created := s.createdAt[s.currentKid]
	return time.Since(created) >= s.cfg.RotationTTL
}

// ======================= BOOTSTRAP / SYNC / ROTATION =======================

func (s *DistributedJWKSService) bootstrap(ctx context.Context) error {
	state, err := s.store.LoadState(ctx)
	if err != nil {
		return fmt.Errorf("jwks load state: %w", err)
	}
	if state != nil && state.CurrentKID != "" {
		if err := s.applyState(*state); err != nil {
			return fmt.Errorf("jwks apply state: %w", err)
		}
		s.log.Printf("jwks: loaded state (current kid=%s)", state.CurrentKID)
		return nil
	}

	// Pas d'état : tenter d'être leader pour créer l'initial
	acq, token, err := s.store.TryAcquireLock(ctx, s.cfg.StateKeyLock, s.cfg.LockTTL)
	if err != nil {
		return fmt.Errorf("jwks acquire lock: %w", err)
	}
	if !acq {
		// quelqu'un d'autre va créer; on attend une sync
		time.Sleep(300 * time.Millisecond)
		return s.syncFromStore(ctx)
	}
	defer func() { _, _ = s.store.ReleaseLock(context.Background(), s.cfg.StateKeyLock, token) }()

	// re-check
	state, _ = s.store.LoadState(ctx)
	if state != nil && state.CurrentKID != "" {
		return s.applyState(*state)
	}

	// créer première clé
	newState, err := s.newStateWithFreshKey()
	if err != nil {
		return err
	}
	if err := s.store.SaveState(ctx, newState); err != nil {
		return fmt.Errorf("jwks save initial state: %w", err)
	}
	return s.applyState(newState)
}

func (s *DistributedJWKSService) syncFromStore(ctx context.Context) error {
	state, err := s.store.LoadState(ctx)
	if err != nil {
		s.log.Printf("jwks: sync load failed: %v", err)
		return err
	}
	if state == nil || state.CurrentKID == "" {
		// rien à sync (premier lancement ?)
		return nil
	}
	if err := s.applyState(*state); err != nil {
		s.log.Printf("jwks: apply state failed: %v", err)
		return err
	}
	return nil
}

func (s *DistributedJWKSService) attemptRotation(ctx context.Context) {
	acq, token, err := s.store.TryAcquireLock(ctx, s.cfg.StateKeyLock, s.cfg.LockTTL)
	if err != nil || !acq {
		return
	}
	defer func() { _, _ = s.store.ReleaseLock(context.Background(), s.cfg.StateKeyLock, token) }()

	state, err := s.store.LoadState(ctx)
	if err != nil {
		s.log.Printf("jwks: rotation load failed: %v", err)
		return
	}

	// 1. Délégation de la vérification de la date
	if s.isRecentlyRotated(state) {
		return
	}

	// 2. Délégation de la préparation de l'état
	merged := s.prepareMergedState(state)

	// 3. Délégation de la génération de clé
	if err := s.generateAndMergeKey(&merged); err != nil {
		s.log.Printf("jwks: %v", err)
		return
	}

	merged = purgeState(merged, s.cfg.RetentionPeriod, s.cfg.MaxKeys)

	if err := s.store.SaveState(ctx, merged); err != nil {
		s.log.Printf("jwks: rotation save failed: %v", err)
		return
	}

	_ = s.applyState(merged)
	s.log.Printf("jwks: rotated (new kid=%s)", merged.CurrentKID)
}

// --- NOUVEAUX HELPERS POUR LA ROTATION ---

func (s *DistributedJWKSService) isRecentlyRotated(state *ports.SharedJWKSState) bool {
	if state == nil || state.CurrentKID == "" {
		return false
	}
	createdStr := state.CreatedAt[state.CurrentKID]
	if createdStr == "" {
		return false
	}
	if created, err := time.Parse(time.RFC3339, createdStr); err == nil {
		return time.Since(created) < s.cfg.RotationTTL
	}
	return false
}

func (s *DistributedJWKSService) prepareMergedState(state *ports.SharedJWKSState) ports.SharedJWKSState {
	if state == nil {
		return ports.SharedJWKSState{Version: 1, Keys: map[string]string{}, CreatedAt: map[string]string{}}
	}
	return ports.SharedJWKSState{
		Version:    state.Version,
		CurrentKID: state.CurrentKID,
		Keys:       state.Keys,
		CreatedAt:  state.CreatedAt,
	}
}

func (s *DistributedJWKSService) generateAndMergeKey(merged *ports.SharedJWKSState) error {
	priv, kid, err := generateRSAKeyAndKID(s.cfg.KeySize)
	if err != nil {
		return fmt.Errorf("rotation generate failed: %w", err)
	}
	pemPriv, err := encodePrivateKeyToPEM(priv)
	if err != nil {
		return fmt.Errorf("encode pem failed: %w", err)
	}

	merged.CurrentKID = kid
	if merged.Keys == nil {
		merged.Keys = map[string]string{}
	}
	if merged.CreatedAt == nil {
		merged.CreatedAt = map[string]string{}
	}
	merged.Keys[kid] = pemPriv
	merged.CreatedAt[kid] = time.Now().UTC().Format(time.RFC3339)
	return nil
}
func (s *DistributedJWKSService) newStateWithFreshKey() (ports.SharedJWKSState, error) {
	priv, kid, err := generateRSAKeyAndKID(s.cfg.KeySize)
	if err != nil {
		return ports.SharedJWKSState{}, err
	}
	pemPriv, err := encodePrivateKeyToPEM(priv)
	if err != nil {
		return ports.SharedJWKSState{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return ports.SharedJWKSState{
		Version:    1,
		CurrentKID: kid,
		Keys:       map[string]string{kid: pemPriv},
		CreatedAt:  map[string]string{kid: now},
	}, nil
}

func (s *DistributedJWKSService) applyState(state ports.SharedJWKSState) error {
	// build RAM cache from shared state (private keys -> public keys)
	if state.Keys == nil || state.CreatedAt == nil {
		return errors.New("invalid jwks state: nil maps")
	}
	if state.CurrentKID == "" {
		return errors.New("invalid jwks state: empty current kid")
	}

	pub := make(map[string]*rsa.PublicKey, len(state.Keys))
	created := make(map[string]time.Time, len(state.CreatedAt))

	var current *rsa.PrivateKey
	for kid, pemPriv := range state.Keys {
		priv, err := decodePEMToPrivateKey(pemPriv)
		if err != nil {
			// si une clé est corrompue, on la skip mais on continue
			s.log.Printf("jwks: skip invalid key kid=%s: %v", kid, err)
			continue
		}
		pub[kid] = &priv.PublicKey

		if tStr, ok := state.CreatedAt[kid]; ok && tStr != "" {
			if t, err := time.Parse(time.RFC3339, tStr); err == nil {
				created[kid] = t
			}
		}
		if kid == state.CurrentKID {
			current = priv
		}
	}

	if current == nil {
		return errors.New("jwks state current key not decodable")
	}

	s.mu.Lock()
	s.current = current
	s.currentKid = state.CurrentKID
	s.publicKeys = pub
	s.createdAt = created
	s.mu.Unlock()

	return nil
}

// ======================= HELPERS (RSA/PEM + PURGE) =======================

func generateRSAKeyAndKID(bits int) (*rsa.PrivateKey, string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, "", err
	}
	kid, err := generateKID()
	if err != nil {
		return nil, "", err
	}
	return priv, kid, nil
}

func generateKID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

func encodePrivateKeyToPEM(priv *rsa.PrivateKey) (string, error) {
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", err
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	return string(pem.EncodeToMemory(block)), nil
}

func decodePEMToPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil || len(block.Bytes) == 0 {
		return nil, errors.New("pem decode failed")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not RSA private key")
	}
	return priv, nil
}

// purgeState : supprime les clés trop vieilles et applique un max keys.
func purgeState(state ports.SharedJWKSState, retention time.Duration, maxKeys int) ports.SharedJWKSState {
	state = ports.SharedJWKSState{
		Version:    state.Version,
		CurrentKID: state.CurrentKID,
		Keys:       cloneMap(state.Keys),
		CreatedAt:  cloneMap(state.CreatedAt),
	}

	// Délégation aux helpers
	applyRetentionPurge(&state, retention)
	applyMaxKeysPurge(&state, maxKeys)

	return state
}

// --- NOUVEAUX HELPERS POUR LA PURGE ---

func applyRetentionPurge(state *ports.SharedJWKSState, retention time.Duration) {
	now := time.Now().UTC()
	for kid, tStr := range state.CreatedAt {
		if kid == state.CurrentKID {
			continue
		}
		if t, err := time.Parse(time.RFC3339, tStr); err == nil && now.Sub(t) > retention {
			delete(state.CreatedAt, kid)
			delete(state.Keys, kid)
		}
	}
}

func applyMaxKeysPurge(state *ports.SharedJWKSState, maxKeys int) {
	if maxKeys <= 0 || len(state.Keys) <= maxKeys {
		return
	}

	type item struct {
		k string
		t time.Time
	}
	var items []item
	for kid, tStr := range state.CreatedAt {
		if t, err := time.Parse(time.RFC3339, tStr); err == nil {
			items = append(items, item{k: kid, t: t})
		}
	}

	// Tri des plus récents aux plus anciens
	sort.Slice(items, func(i, j int) bool { return items[i].t.After(items[j].t) })

	keep := map[string]bool{state.CurrentKID: true}
	for _, it := range items {
		if len(keep) >= maxKeys {
			break
		}
		keep[it.k] = true
	}

	for kid := range state.Keys {
		if !keep[kid] {
			delete(state.Keys, kid)
			delete(state.CreatedAt, kid)
		}
	}
}
func cloneMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// encodeRSAKeyParams extrait et encode le Modulus (N) et l'Exponent (E)
// au format Base64 RawURL (sans padding), requis par la spécification RFC 7518.
func encodeRSAKeyParams(pub *rsa.PublicKey) (n string, e string) {
	n = base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e = base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	return n, e
}
