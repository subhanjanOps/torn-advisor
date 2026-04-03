package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// BoosterRule checks if the player can use a booster.
type BoosterRule struct {
	Priority int
}

func (r BoosterRule) Evaluate(state domain.PlayerState) *domain.Action {
	if state.BoosterCooldown == 0 {
		return &domain.Action{
			Name:        "Use Booster",
			Description: "Booster cooldown is ready — use a stat booster.",
			Priority:    r.Priority,
			Category:    domain.CategoryBooster,
		}
	}
	return nil
}
