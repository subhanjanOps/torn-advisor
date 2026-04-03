package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// HospitalRule checks if the player's life is critically low.
type HospitalRule struct{}

func (r HospitalRule) Evaluate(state domain.PlayerState) *domain.Action {
	if state.LifeMax > 0 && state.Life < state.LifeMax/2 {
		return &domain.Action{
			Name:        "Heal Up",
			Description: "Life is below 50% — heal before taking any action.",
			Priority:    98,
			Category:    domain.CategoryHospital,
		}
	}
	return nil
}
