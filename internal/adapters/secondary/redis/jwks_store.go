package redis

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yvan/nexora-core/internal/core/ports" // ✅ Import crucial
)

// RedisJWKSStore implémente ports.JWKSStore
type RedisJWKSStore struct {
	client        *redis.Client
	stateKey      string
	encryptionKey []byte
}

func NewRedisJWKSStore(client *redis.Client, stateKey string, secretKey string) (*RedisJWKSStore, error) {
	if client == nil {
		return nil, errors.New("redis client is nil")
	}
	if stateKey == "" {
		return nil, errors.New("stateKey is empty")
	}

	keyBytes := []byte(secretKey)
	if len(keyBytes) != 16 && len(keyBytes) != 24 && len(keyBytes) != 32 {
		return nil, errors.New("encryption key must be exactly 16, 24, or 32 bytes for AES")
	}

	return &RedisJWKSStore{
		client:        client,
		stateKey:      stateKey,
		encryptionKey: keyBytes,
	}, nil
}

// ======================= SAVE & LOAD (CORE LOGIC) =======================

func (r *RedisJWKSStore) SaveState(ctx context.Context, state ports.SharedJWKSState) error {
	// 1. Préparation de l'état chiffré pour le réseau
	encryptedState := ports.SharedJWKSState{
		Version:    state.Version,
		CurrentKID: state.CurrentKID,
		Keys:       make(map[string]string, len(state.Keys)),
		CreatedAt:  make(map[string]string, len(state.CreatedAt)),
	}

	for k, v := range state.CreatedAt {
		encryptedState.CreatedAt[k] = v
	}

	// 2. Chiffrement AES-GCM des clés privées PEM
	for kid, pem := range state.Keys {
		if kid == "" {
			continue
		}
		enc, err := encryptAESGCMRaw([]byte(pem), r.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt key %s: %w", kid, err)
		}
		encryptedState.Keys[kid] = enc
	}

	data, err := json.Marshal(encryptedState)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}

	return r.client.Set(ctx, r.stateKey, data, 0).Err()
}

func (r *RedisJWKSStore) LoadState(ctx context.Context) (*ports.SharedJWKSState, error) {
	data, err := r.client.Get(ctx, r.stateKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var state ports.SharedJWKSState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %w", err)
	}

	// 3. Déchiffrement à la volée vers la RAM
	for kid, enc := range state.Keys {
		dec, err := decryptAESGCMRaw(enc, r.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("decryption failed for kid %s (check encryption key): %w", kid, err)
		}
		state.Keys[kid] = string(dec)
	}

	return &state, nil
}

// ======================= DISTRIBUTED LOCK (REDIS) =======================

func (r *RedisJWKSStore) TryAcquireLock(ctx context.Context, lockName string, ttl time.Duration) (bool, string, error) {
	token, _ := newLockToken()
	acquired, err := r.client.SetNX(ctx, lockName, token, ttl).Result()
	if err != nil {
		return false, "", err
	}
	return acquired, token, nil
}

var luaReleaseLock = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`)

func (r *RedisJWKSStore) ReleaseLock(ctx context.Context, lockName, token string) (bool, error) {
	res, err := luaReleaseLock.Run(ctx, r.client, []string{lockName}, token).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// ======================= CRYPTO HELPERS =======================

func encryptAESGCMRaw(plaintext []byte, key []byte) (string, error) {
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func decryptAESGCMRaw(base64Ciphertext string, key []byte) ([]byte, error) {
	ciphertext, _ := base64.RawStdEncoding.DecodeString(base64Ciphertext)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	ns := gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, errors.New("ciphertext too short")
	}

	nonce, body := ciphertext[:ns], ciphertext[ns:]
	return gcm.Open(nil, nonce, body, nil)
}

func newLockToken() (string, error) {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b), nil
}

// SHA256Hasher implémente l'interface ports.TokenHasher
type SHA256Hasher struct{}

func NewSHA256Hasher() *SHA256Hasher {
	return &SHA256Hasher{}
}

// Hash transforme un token brut en empreinte SHA256
func (h *SHA256Hasher) Hash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
