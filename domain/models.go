package domain

import "context"

// PlayerState represents the player's current state in Torn City.
// This struct is the main input to the engine — all rules evaluate against it.
type PlayerState struct {
	Energy          int
	EnergyMax       int
	Nerve           int
	NerveMax        int
	Happy           int
	Life            int
	XanaxCooldown   int
	BoosterCooldown int
	TravelCooldown  int
	MedicalCooldown int
	Addiction       int
	Strength        int64
	Defense         int64
	Speed           int64
	Dexterity       int64
	WarActive       bool
	ChainActive     bool
}

// Category classifies the type of action.
type Category string

const (
	CategoryGym    Category = "gym"
	CategoryCrime  Category = "crime"
	CategoryTravel Category = "travel"
	CategoryWar    Category = "war"
	CategoryRehab  Category = "rehab"
	CategoryDrug   Category = "drug"
)

// Action represents a recommended action for the player.
type Action struct {
	Name        string
	Description string
	Priority    int
	Category    Category
}

// Rule defines the interface that all advisor rules must implement.
// Each rule checks a condition against the player state and returns
// an Action if the condition is met, or nil if not applicable.
type Rule interface {
	Evaluate(state PlayerState) *Action
}

// StateProvider abstracts the data source that supplies player state.
// The engine depends on this interface rather than any concrete SDK or API.
type StateProvider interface {
	FetchPlayerState(ctx context.Context) (PlayerState, error)
}
