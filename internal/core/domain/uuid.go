package domain

import "github.com/google/uuid"

// NewUUID génère un identifiant unique universel de version 4.
func NewUUID() string {
	return uuid.New().String()
}
