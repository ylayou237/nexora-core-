package domain_test

import (
	"time"
)

// FakeClock est un mock de l'interface domain.Clock.
// Il permet de contrôler le temps de manière déterministe.
type FakeClock struct {
	currentTime time.Time
}

func NewFakeClock() *FakeClock {
	// On fixe une date de départ arbitraire mais valide
	return &FakeClock{
		currentTime: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}
}

func (f *FakeClock) Now() time.Time {
	return f.currentTime
}

// Advance permet d'avancer le temps (ex: simuler une expiration)
func (f *FakeClock) Advance(d time.Duration) {
	f.currentTime = f.currentTime.Add(d)
}
