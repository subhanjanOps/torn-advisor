package rules

import (
	"github.com/subhanjanOps/torn-advisor/config"
	"github.com/subhanjanOps/torn-advisor/domain"
)

// DefaultRules returns the standard set of rules using built-in priorities.
func DefaultRules() []domain.Rule {
	return DefaultRulesWithConfig(config.DefaultPriorities())
}

// DefaultRulesWithConfig returns the standard set of rules with configurable priorities.
func DefaultRulesWithConfig(cfg config.RulePriorities) []domain.Rule {
	return []domain.Rule{
		HospitalRule{Priority: cfg.Hospital},
		ChainRule{Priority: cfg.Chain},
		WarRule{Priority: cfg.War},
		XanaxRule{Priority: cfg.Xanax},
		RehabRule{Priority: cfg.Rehab},
		GymRule{Priority: cfg.Gym},
		CrimeRule{Priority: cfg.Crime},
		TravelRule{Priority: cfg.Travel},
		BoosterRule{Priority: cfg.Booster},
	}
}
