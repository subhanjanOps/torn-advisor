package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// CrimeRule checks if the player's nerve is at maximum.
type CrimeRule struct{}

func (r CrimeRule) Evaluate(state domain.PlayerState) *domain.Action {
	if state.NerveMax > 0 && state.Nerve == state.NerveMax {
		return &domain.Action{
			Name:        "Do Crimes",
			Description: "Nerve is full — commit crimes before it's wasted.",
			Priority:    70,
			Category:    domain.CategoryCrime,
		}
	}
	return nil
}
