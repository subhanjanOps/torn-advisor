package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// RulePriorities maps rule names to their priority values.
type RulePriorities struct {
	Hospital int `json:"hospital"`
	Chain    int `json:"chain"`
	War      int `json:"war"`
	Xanax    int `json:"xanax"`
	Rehab    int `json:"rehab"`
	Gym      int `json:"gym"`
	Crime    int `json:"crime"`
	Travel   int `json:"travel"`
	Booster  int `json:"booster"`
}

// DefaultPriorities returns the built-in priority values.
func DefaultPriorities() RulePriorities {
	return RulePriorities{
		Hospital: 98,
		Chain:    97,
		War:      95,
		Xanax:    90,
		Rehab:    85,
		Gym:      80,
		Crime:    70,
		Travel:   60,
		Booster:  55,
	}
}

// LoadPriorities reads priorities from a JSON file.
// Any field not specified in the file retains its default value.
func LoadPriorities(path string) (RulePriorities, error) {
	p := DefaultPriorities()

	data, err := os.ReadFile(path)
	if err != nil {
		return p, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("parsing config: %w", err)
	}

	return p, nil
}
