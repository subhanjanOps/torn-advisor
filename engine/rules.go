package engine

// WarRule checks if a war is active and recommends saving energy.
type WarRule struct{}

func (r WarRule) Evaluate(state PlayerState) *Action {
	if state.WarActive {
		return &Action{
			Name:        "Save Energy for War",
			Description: "War is active — conserve energy for war targets.",
			Priority:    95,
			Category:    CategoryWar,
		}
	}
	return nil
}

// XanaxRule checks if the player can take Xanax.
type XanaxRule struct{}

func (r XanaxRule) Evaluate(state PlayerState) *Action {
	if state.XanaxCooldown == 0 {
		return &Action{
			Name:        "Take Xanax",
			Description: "Xanax cooldown is ready — take Xanax for an energy boost.",
			Priority:    90,
			Category:    CategoryDrug,
		}
	}
	return nil
}

// RehabRule checks if the player's addiction is above a threshold.
type RehabRule struct{}

const addictionThreshold = 50

func (r RehabRule) Evaluate(state PlayerState) *Action {
	if state.Addiction > addictionThreshold {
		return &Action{
			Name:        "Rehab",
			Description: "Addiction level is high — visit rehab to reduce it.",
			Priority:    85,
			Category:    CategoryRehab,
		}
	}
	return nil
}

// GymRule checks if the player has energy and enough happy to train.
type GymRule struct{}

const happyTrainThreshold = 4000

func (r GymRule) Evaluate(state PlayerState) *Action {
	if state.Energy > 0 && state.Happy > happyTrainThreshold {
		return &Action{
			Name:        "Train at Gym",
			Description: "Energy and happiness are sufficient — train your stats.",
			Priority:    80,
			Category:    CategoryGym,
		}
	}
	return nil
}

// CrimeRule checks if the player's nerve is at maximum.
type CrimeRule struct{}

func (r CrimeRule) Evaluate(state PlayerState) *Action {
	if state.NerveMax > 0 && state.Nerve == state.NerveMax {
		return &Action{
			Name:        "Do Crimes",
			Description: "Nerve is full — commit crimes before it's wasted.",
			Priority:    70,
			Category:    CategoryCrime,
		}
	}
	return nil
}

// TravelRule checks if the player can travel.
type TravelRule struct{}

func (r TravelRule) Evaluate(state PlayerState) *Action {
	if state.TravelCooldown == 0 {
		return &Action{
			Name:        "Fly Abroad",
			Description: "Travel cooldown is clear — fly for profit.",
			Priority:    60,
			Category:    CategoryTravel,
		}
	}
	return nil
}

// DefaultRules returns the standard set of rules for the advisor engine.
func DefaultRules() []Rule {
	return []Rule{
		WarRule{},
		XanaxRule{},
		RehabRule{},
		GymRule{},
		CrimeRule{},
		TravelRule{},
	}
}
