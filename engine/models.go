package engine

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
