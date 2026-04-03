package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// XanaxRule checks if the player can take Xanax.
type XanaxRule struct{}

func (r XanaxRule) Evaluate(state domain.PlayerState) *domain.Action {
	if state.XanaxCooldown == 0 {
		return &domain.Action{
			Name:        "Take Xanax",
			Description: "Xanax cooldown is ready — take Xanax for an energy boost.",
			Priority:    90,
			Category:    domain.CategoryDrug,
		}
	}
	return nil
}
