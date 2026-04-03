package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// ChainRule checks if a faction chain is active and the timeout is approaching.
type ChainRule struct{}

const chainTimeoutThreshold = 60 // seconds

func (r ChainRule) Evaluate(state domain.PlayerState) *domain.Action {
	if state.ChainActive {
		return &domain.Action{
			Name:        "Continue Chain",
			Description: "Chain is active — hit a target to keep it alive.",
			Priority:    97,
			Category:    domain.CategoryChain,
		}
	}
	return nil
}
