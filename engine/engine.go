package engine

import "github.com/subhanjanOps/torn-advisor/domain"

// Engine evaluates all registered rules against a player state
// and produces a sorted action plan.
type Engine struct {
	rules []Rule
}

// NewEngine creates an Engine initialized with the given rules.
func NewEngine(rules []Rule) *Engine {
	return &Engine{rules: rules}
}

// Run evaluates every rule against the given state and returns
// an ordered list of recommended actions (highest priority first).
func (e *Engine) Run(state domain.PlayerState) []domain.Action {
	actions := make([]*domain.Action, 0, len(e.rules))
	for _, r := range e.rules {
		actions = append(actions, r.Evaluate(state))
	}
	return BuildPlan(actions)
}
