package engine

import (
	"sort"

	"github.com/subhanjanOps/torn-advisor/domain"
)

// BuildPlan takes a slice of actions, removes nil entries, and sorts
// them by priority in descending order (highest priority first).
func BuildPlan(actions []*domain.Action) []domain.Action {
	plan := make([]domain.Action, 0, len(actions))
	for _, a := range actions {
		if a != nil {
			plan = append(plan, *a)
		}
	}

	sort.Slice(plan, func(i, j int) bool {
		return plan[i].Priority > plan[j].Priority
	})

	return plan
}
