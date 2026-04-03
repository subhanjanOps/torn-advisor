package rules

import "github.com/subhanjanOps/torn-advisor/domain"

// DefaultRules returns the standard set of rules for the advisor engine.
func DefaultRules() []domain.Rule {
	return []domain.Rule{
		WarRule{},
		XanaxRule{},
		RehabRule{},
		GymRule{},
		CrimeRule{},
		TravelRule{},
	}
}
