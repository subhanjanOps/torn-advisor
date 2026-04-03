package engine

// Rule defines the interface that all advisor rules must implement.
// Each rule checks a condition against the player state and returns
// an Action if the condition is met, or nil if not applicable.
type Rule interface {
	Evaluate(state PlayerState) *Action
}
