package domain

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"time"
)

// ======================= STRUCTS INTERNE =======================

// RSAKey représente une paire de clés asymétriques avec son identifiant unique (kid)
type RSAKey struct {
	ID         string
	PrivateKey *rsa.PrivateKey
	CreatedAt  time.Time
}

// PublicKey retourne la partie publique de la paire
func (k *RSAKey) PublicKey() *rsa.PublicKey {
	return &k.PrivateKey.PublicKey
}

// ======================= STRUCTS PUBLIQUES JWKS =======================

// JWKSKey est exposée pour les clients (JSON Web Key Set)
type JWKSKey struct {
	Kid string `json:"kid"` // identifiant unique de la clé
	Kty string `json:"kty"` // type de clé ("RSA")
	Alg string `json:"alg"` // algorithme ("RS256")
	Use string `json:"use"` // usage ("sig")
	N   string `json:"n"`   // modulus base64url
	E   string `json:"e"`   // exponent base64url
}

// JWKS représente l'ensemble des clés publiques exposées
type JWKS struct {
	Keys []JWKSKey `json:"keys"`
}

// ======================= HELPER =======================

// ToJWKSKey convertit une RSAKey en JWKSKey
func (k *RSAKey) ToJWKSKey() JWKSKey {
	pub := k.PublicKey()
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())

	return JWKSKey{
		Kid: k.ID,
		Kty: "RSA",
		Alg: "RS256",
		Use: "sig",
		N:   n,
		E:   e,
	}
}
