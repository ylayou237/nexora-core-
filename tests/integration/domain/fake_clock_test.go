package domain_test

import "time"

// FakeClock est utilisé pour tester les entités qui dépendent du temps
type FakeClock struct {
	now time.Time
}

// Now retourne l'heure courante simulée
func (f *FakeClock) Now() time.Time {
	return f.now
}

// Advance avance le temps simulé de la durée spécifiée
func (f *FakeClock) Advance(d time.Duration) {
	f.now = f.now.Add(d)
}
