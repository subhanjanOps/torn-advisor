package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// TravelRule checks if the player can travel.
type TravelRule struct{}

func (r TravelRule) Evaluate(state domain.PlayerState) *domain.Action {
	if state.TravelCooldown == 0 {
		return &domain.Action{
			Name:        "Fly Abroad",
			Description: "Travel cooldown is clear — fly for profit.",
			Priority:    60,
			Category:    domain.CategoryTravel,
		}
	}
	return nil
}
