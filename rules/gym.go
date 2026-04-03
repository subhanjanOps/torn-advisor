package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// GymRule checks if the player has energy and enough happy to train.
type GymRule struct{}

const happyTrainThreshold = 4000

func (r GymRule) Evaluate(state domain.PlayerState) *domain.Action {
	if state.Energy > 0 && state.Happy > happyTrainThreshold {
		return &domain.Action{
			Name:        "Train at Gym",
			Description: "Energy and happiness are sufficient — train your stats.",
			Priority:    80,
			Category:    domain.CategoryGym,
		}
	}
	return nil
}
