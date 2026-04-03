package cache

import (
	"context"
	"sync"
	"time"

	"github.com/subhanjanOps/torn-advisor/domain"
)

// Provider wraps a StateProvider with time-based caching.
type Provider struct {
	inner domain.StateProvider
	ttl   time.Duration

	mu      sync.RWMutex
	state   domain.PlayerState
	fetched time.Time
	valid   bool
}

// NewProvider creates a caching wrapper around the given provider.
// Cached results are reused for the specified TTL duration.
func NewProvider(inner domain.StateProvider, ttl time.Duration) *Provider {
	return &Provider{inner: inner, ttl: ttl}
}

// FetchPlayerState returns a cached state if still fresh, otherwise fetches anew.
func (p *Provider) FetchPlayerState(ctx context.Context) (domain.PlayerState, error) {
	p.mu.RLock()
	if p.valid && time.Since(p.fetched) < p.ttl {
		state := p.state
		p.mu.RUnlock()
		return state, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check: another goroutine may have refreshed while we waited for the lock.
	if p.valid && time.Since(p.fetched) < p.ttl {
		return p.state, nil
	}

	state, err := p.inner.FetchPlayerState(ctx)
	if err != nil {
		return state, err
	}

	p.state = state
	p.fetched = time.Now()
	p.valid = true

	return state, nil
}

// Invalidate forces the next call to fetch fresh data.
func (p *Provider) Invalidate() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.valid = false
}
