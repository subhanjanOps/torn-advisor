package torn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/subhanjanOps/torn-advisor/domain"
	"github.com/subhanjanOps/tornSDK/client"
	"github.com/subhanjanOps/tornSDK/user"
)

// UserAPI defines the subset of the tornSDK user service used by the provider.
// This enables testing with a mock implementation.
type UserAPI interface {
	GetBars(ctx context.Context) (*user.Bars, error)
	GetBattleStats(ctx context.Context) (*user.BattleStats, error)
	GetMyCooldowns(ctx context.Context, query url.Values) (json.RawMessage, error)
}

// Provider converts tornSDK data into a PlayerState.
// It implements the domain.StateProvider interface.
type Provider struct {
	user UserAPI
}

// NewProvider creates a Provider backed by the given SDK client.
func NewProvider(sdk *client.Client) *Provider {
	return &Provider{user: sdk.User}
}

// NewProviderFromAPI creates a Provider from any UserAPI implementation.
// This is useful for testing with mocks.
func NewProviderFromAPI(api UserAPI) *Provider {
	return &Provider{user: api}
}

// cooldowns mirrors the JSON shape returned by the Torn API cooldowns endpoint.
type cooldowns struct {
	Cooldowns struct {
		Drug    int `json:"drug"`
		Booster int `json:"booster"`
		Medical int `json:"medical"`
	} `json:"cooldowns"`
}

// FetchPlayerState gathers data from the Torn API and maps it into a PlayerState.
func (p *Provider) FetchPlayerState(ctx context.Context) (domain.PlayerState, error) {
	var state domain.PlayerState

	// Fetch bars (energy, nerve, happy, life, chain).
	bars, err := p.user.GetBars(ctx)
	if err != nil {
		return state, fmt.Errorf("fetching bars: %w", err)
	}

	state.Energy = bars.Energy.Current
	state.EnergyMax = bars.Energy.Maximum
	state.Nerve = bars.Nerve.Current
	state.NerveMax = bars.Nerve.Maximum
	state.Happy = bars.Happy.Current
	state.Life = bars.Life.Current
	state.LifeMax = bars.Life.Maximum

	if bars.Chain != nil {
		state.ChainActive = bars.Chain.Current > 0
	}

	// Fetch battle stats.
	stats, err := p.user.GetBattleStats(ctx)
	if err != nil {
		return state, fmt.Errorf("fetching battle stats: %w", err)
	}

	state.Strength = stats.Strength.Value
	state.Defense = stats.Defense.Value
	state.Speed = stats.Speed.Value
	state.Dexterity = stats.Dexterity.Value

	// Fetch cooldowns (raw endpoint).
	raw, err := p.user.GetMyCooldowns(ctx, url.Values{})
	if err != nil {
		return state, fmt.Errorf("fetching cooldowns: %w", err)
	}

	var cd cooldowns
	if err := json.Unmarshal(raw, &cd); err != nil {
		return state, fmt.Errorf("parsing cooldowns: %w", err)
	}

	state.XanaxCooldown = cd.Cooldowns.Drug
	state.BoosterCooldown = cd.Cooldowns.Booster
	state.MedicalCooldown = cd.Cooldowns.Medical

	return state, nil
}
