package torn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/subhanjanOps/torn-advisor/domain"
	"github.com/subhanjanOps/tornSDK/client"
)

// Provider converts tornSDK data into a PlayerState.
// It implements the engine.StateProvider interface.
type Provider struct {
	sdk *client.Client
}

// NewProvider creates a Provider backed by the given SDK client.
func NewProvider(sdk *client.Client) *Provider {
	return &Provider{sdk: sdk}
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
	bars, err := p.sdk.User.GetBars(ctx)
	if err != nil {
		return state, fmt.Errorf("fetching bars: %w", err)
	}

	state.Energy = bars.Energy.Current
	state.EnergyMax = bars.Energy.Maximum
	state.Nerve = bars.Nerve.Current
	state.NerveMax = bars.Nerve.Maximum
	state.Happy = bars.Happy.Current
	state.Life = bars.Life.Current

	if bars.Chain != nil {
		state.ChainActive = bars.Chain.Current > 0
	}

	// Fetch battle stats.
	stats, err := p.sdk.User.GetBattleStats(ctx)
	if err != nil {
		return state, fmt.Errorf("fetching battle stats: %w", err)
	}

	state.Strength = stats.Strength.Value
	state.Defense = stats.Defense.Value
	state.Speed = stats.Speed.Value
	state.Dexterity = stats.Dexterity.Value

	// Fetch cooldowns (raw endpoint).
	raw, err := p.sdk.User.GetMyCooldowns(ctx, url.Values{})
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
