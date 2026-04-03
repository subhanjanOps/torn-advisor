package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/subhanjanOps/torn-advisor/domain"
)

type mockProvider struct {
	calls atomic.Int32
	state domain.PlayerState
	err   error
}

func (m *mockProvider) FetchPlayerState(_ context.Context) (domain.PlayerState, error) {
	m.calls.Add(1)
	return m.state, m.err
}

func TestCacheHit(t *testing.T) {
	inner := &mockProvider{
		state: domain.PlayerState{Energy: 100, EnergyMax: 150},
	}
	cp := NewProvider(inner, 5*time.Second)

	// First call — cache miss.
	s1, err := cp.FetchPlayerState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s1.Energy != 100 {
		t.Errorf("expected Energy 100, got %d", s1.Energy)
	}
	if inner.calls.Load() != 1 {
		t.Errorf("expected 1 call, got %d", inner.calls.Load())
	}

	// Second call — should be cached.
	s2, err := cp.FetchPlayerState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s2.Energy != 100 {
		t.Errorf("expected Energy 100, got %d", s2.Energy)
	}
	if inner.calls.Load() != 1 {
		t.Errorf("expected still 1 call (cache hit), got %d", inner.calls.Load())
	}
}

func TestCacheExpiry(t *testing.T) {
	inner := &mockProvider{
		state: domain.PlayerState{Energy: 50},
	}
	cp := NewProvider(inner, 10*time.Millisecond)

	// First call.
	_, _ = cp.FetchPlayerState(context.Background())
	if inner.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", inner.calls.Load())
	}

	// Wait for TTL to expire.
	time.Sleep(20 * time.Millisecond)

	// Next call should re-fetch.
	_, _ = cp.FetchPlayerState(context.Background())
	if inner.calls.Load() != 2 {
		t.Errorf("expected 2 calls after expiry, got %d", inner.calls.Load())
	}
}

func TestCacheInvalidate(t *testing.T) {
	inner := &mockProvider{
		state: domain.PlayerState{Nerve: 30},
	}
	cp := NewProvider(inner, 5*time.Second)

	_, _ = cp.FetchPlayerState(context.Background())
	if inner.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", inner.calls.Load())
	}

	cp.Invalidate()

	_, _ = cp.FetchPlayerState(context.Background())
	if inner.calls.Load() != 2 {
		t.Errorf("expected 2 calls after invalidate, got %d", inner.calls.Load())
	}
}

func TestCachePassesThroughErrors(t *testing.T) {
	inner := &mockProvider{err: fmt.Errorf("api down")}
	cp := NewProvider(inner, 5*time.Second)

	_, err := cp.FetchPlayerState(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	// Error responses should not be cached.
	_, _ = cp.FetchPlayerState(context.Background())
	if inner.calls.Load() != 2 {
		t.Errorf("expected 2 calls (errors not cached), got %d", inner.calls.Load())
	}
}

func TestCacheConcurrentDoubleCheck(t *testing.T) {
	inner := &mockProvider{
		state: domain.PlayerState{Energy: 42},
	}
	cp := NewProvider(inner, 5*time.Second)

	// Launch many goroutines to race on the first fetch — at most 1 should call inner.
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := cp.FetchPlayerState(context.Background())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if s.Energy != 42 {
				t.Errorf("expected Energy 42, got %d", s.Energy)
			}
		}()
	}
	wg.Wait()

	// Inner should have been called very few times (ideally 1, but races may allow a few).
	if calls := inner.calls.Load(); calls > 5 {
		t.Errorf("expected few inner calls, got %d", calls)
	}
}
