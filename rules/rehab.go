package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// RehabRule checks if the player's addiction is above a threshold.
type RehabRule struct {
	Priority int
}

const addictionThreshold = 50

func (r RehabRule) Evaluate(state domain.PlayerState) *domain.Action {
	if state.Addiction > addictionThreshold {
		return &domain.Action{
			Name:        "Rehab",
			Description: "Addiction level is high — visit rehab to reduce it.",
			Priority:    r.Priority,
			Category:    domain.CategoryRehab,
		}
	}
	return nil
}
