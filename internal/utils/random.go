package utils

import (
	"math/rand"
	"sync"
)

// RNG wraps deterministic pseudo-random operations.
type RNG interface {
	Intn(max int) int
	CoinFlip() bool
}

type deterministicRNG struct {
	mu sync.Mutex
	r  *rand.Rand
}

// NewDeterministicRNG creates a deterministic RNG from a fixed seed.
func NewDeterministicRNG(seed int64) RNG {
	return &deterministicRNG{r: rand.New(rand.NewSource(seed))}
}

func (d *deterministicRNG) Intn(max int) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.r.Intn(max)
}

func (d *deterministicRNG) CoinFlip() bool {
	return d.Intn(2) == 1
}
