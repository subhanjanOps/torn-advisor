package torn

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/subhanjanOps/tornSDK/user"
)

// mockUserAPI is a test double implementing UserAPI.
type mockUserAPI struct {
	bars      *user.Bars
	barsErr   error
	stats     *user.BattleStats
	statsErr  error
	cooldowns json.RawMessage
	cdErr     error
}

func (m *mockUserAPI) GetBars(_ context.Context) (*user.Bars, error) {
	return m.bars, m.barsErr
}

func (m *mockUserAPI) GetBattleStats(_ context.Context) (*user.BattleStats, error) {
	return m.stats, m.statsErr
}

func (m *mockUserAPI) GetMyCooldowns(_ context.Context, _ url.Values) (json.RawMessage, error) {
	return m.cooldowns, m.cdErr
}

func defaultMock() *mockUserAPI {
	return &mockUserAPI{
		bars: &user.Bars{
			Energy: user.Bar{Current: 150, Maximum: 300},
			Nerve:  user.Bar{Current: 50, Maximum: 60},
			Happy:  user.Bar{Current: 8000, Maximum: 10000},
			Life:   user.Bar{Current: 7500, Maximum: 7500},
			Chain:  &user.Chain{Current: 5, Max: 10},
		},
		stats: &user.BattleStats{
			Strength:  user.BattleStat{Value: 1000000},
			Defense:   user.BattleStat{Value: 2000000},
			Speed:     user.BattleStat{Value: 500000},
			Dexterity: user.BattleStat{Value: 750000},
		},
		cooldowns: json.RawMessage(`{"cooldowns":{"drug":0,"booster":120,"medical":0}}`),
	}
}

func TestFetchPlayerState_Success(t *testing.T) {
	mock := defaultMock()
	provider := NewProviderFromAPI(mock)

	state, err := provider.FetchPlayerState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bars
	if state.Energy != 150 {
		t.Errorf("Energy: want 150, got %d", state.Energy)
	}
	if state.EnergyMax != 300 {
		t.Errorf("EnergyMax: want 300, got %d", state.EnergyMax)
	}
	if state.Nerve != 50 {
		t.Errorf("Nerve: want 50, got %d", state.Nerve)
	}
	if state.NerveMax != 60 {
		t.Errorf("NerveMax: want 60, got %d", state.NerveMax)
	}
	if state.Happy != 8000 {
		t.Errorf("Happy: want 8000, got %d", state.Happy)
	}
	if state.Life != 7500 {
		t.Errorf("Life: want 7500, got %d", state.Life)
	}

	// Chain
	if !state.ChainActive {
		t.Error("ChainActive: want true, got false")
	}

	// Battle stats
	if state.Strength != 1000000 {
		t.Errorf("Strength: want 1000000, got %d", state.Strength)
	}
	if state.Defense != 2000000 {
		t.Errorf("Defense: want 2000000, got %d", state.Defense)
	}
	if state.Speed != 500000 {
		t.Errorf("Speed: want 500000, got %d", state.Speed)
	}
	if state.Dexterity != 750000 {
		t.Errorf("Dexterity: want 750000, got %d", state.Dexterity)
	}

	// Cooldowns
	if state.XanaxCooldown != 0 {
		t.Errorf("XanaxCooldown: want 0, got %d", state.XanaxCooldown)
	}
	if state.BoosterCooldown != 120 {
		t.Errorf("BoosterCooldown: want 120, got %d", state.BoosterCooldown)
	}
	if state.MedicalCooldown != 0 {
		t.Errorf("MedicalCooldown: want 0, got %d", state.MedicalCooldown)
	}
}

func TestFetchPlayerState_ChainNil(t *testing.T) {
	mock := defaultMock()
	mock.bars.Chain = nil
	provider := NewProviderFromAPI(mock)

	state, err := provider.FetchPlayerState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.ChainActive {
		t.Error("ChainActive: want false when chain is nil")
	}
}

func TestFetchPlayerState_ChainZero(t *testing.T) {
	mock := defaultMock()
	mock.bars.Chain = &user.Chain{Current: 0}
	provider := NewProviderFromAPI(mock)

	state, err := provider.FetchPlayerState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.ChainActive {
		t.Error("ChainActive: want false when chain current is 0")
	}
}

func TestFetchPlayerState_BarsError(t *testing.T) {
	mock := defaultMock()
	mock.barsErr = errors.New("api timeout")
	provider := NewProviderFromAPI(mock)

	_, err := provider.FetchPlayerState(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, mock.barsErr) {
		t.Errorf("expected wrapped bars error, got: %v", err)
	}
}

func TestFetchPlayerState_StatsError(t *testing.T) {
	mock := defaultMock()
	mock.statsErr = errors.New("stats failed")
	provider := NewProviderFromAPI(mock)

	_, err := provider.FetchPlayerState(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, mock.statsErr) {
		t.Errorf("expected wrapped stats error, got: %v", err)
	}
}

func TestFetchPlayerState_CooldownsError(t *testing.T) {
	mock := defaultMock()
	mock.cdErr = errors.New("cooldowns unavailable")
	provider := NewProviderFromAPI(mock)

	_, err := provider.FetchPlayerState(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, mock.cdErr) {
		t.Errorf("expected wrapped cooldowns error, got: %v", err)
	}
}

func TestFetchPlayerState_CooldownsBadJSON(t *testing.T) {
	mock := defaultMock()
	mock.cooldowns = json.RawMessage(`{invalid`)
	provider := NewProviderFromAPI(mock)

	_, err := provider.FetchPlayerState(context.Background())
	if err == nil {
		t.Fatal("expected error on bad JSON, got nil")
	}
}
