package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// WarRule checks if a war is active and recommends saving energy.
type WarRule struct {
	Priority int
}

func (r WarRule) Evaluate(state domain.PlayerState) *domain.Action {
	if state.WarActive {
		return &domain.Action{
			Name:        "Save Energy for War",
			Description: "War is active — conserve energy for war targets.",
			Priority:    r.Priority,
			Category:    domain.CategoryWar,
		}
	}
	return nil
}
