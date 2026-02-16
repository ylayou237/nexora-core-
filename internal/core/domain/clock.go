package domain

import "time"

// --- Interface ---

// Clock abstrait la source de temps pour garantir la testabilité déterministe.
type Clock interface {
	Now() time.Time
}

// --- Real Implementation (Production) ---

type RealClock struct{}

func NewRealClock() RealClock {
	return RealClock{}
}

// Now retourne l'heure actuelle en UTC.
// CRITIQUE : Toujours utiliser UTC pour éviter les bugs de Timezone/DST.
func (r RealClock) Now() time.Time {
	return time.Now().UTC()
}

// --- Mock Implementation (Tests) ---

// FakeClock permet de figer le temps ou de l'avancer manuellement dans les tests.
type FakeClock struct {
	currentTime time.Time
}

func NewFakeClock(t time.Time) *FakeClock {
	return &FakeClock{
		currentTime: t.UTC(),
	}
}

func (f *FakeClock) Now() time.Time {
	return f.currentTime
}

// Advance permet de simuler le passage du temps (ex: "avancer de 1 heure")
func (f *FakeClock) Advance(d time.Duration) {
	f.currentTime = f.currentTime.Add(d)
}

// Set permet de sauter à une date précise
func (f *FakeClock) Set(t time.Time) {
	f.currentTime = t.UTC()
}
