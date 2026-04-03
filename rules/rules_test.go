package rules

import (
	"testing"

	"github.com/subhanjanOps/torn-advisor/domain"
)

// --- WarRule ---

func TestWarRule_Active(t *testing.T) {
	rule := WarRule{}
	state := domain.PlayerState{WarActive: true}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 95 {
		t.Errorf("expected priority 95, got %d", action.Priority)
	}
	if action.Category != domain.CategoryWar {
		t.Errorf("expected category %q, got %q", domain.CategoryWar, action.Category)
	}
}

func TestWarRule_Inactive(t *testing.T) {
	rule := WarRule{}
	state := domain.PlayerState{WarActive: false}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

// --- XanaxRule ---

func TestXanaxRule_Ready(t *testing.T) {
	rule := XanaxRule{}
	state := domain.PlayerState{XanaxCooldown: 0}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 90 {
		t.Errorf("expected priority 90, got %d", action.Priority)
	}
	if action.Category != domain.CategoryDrug {
		t.Errorf("expected category %q, got %q", domain.CategoryDrug, action.Category)
	}
}

func TestXanaxRule_OnCooldown(t *testing.T) {
	rule := XanaxRule{}
	state := domain.PlayerState{XanaxCooldown: 300}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

// --- RehabRule ---

func TestRehabRule_HighAddiction(t *testing.T) {
	rule := RehabRule{}
	state := domain.PlayerState{Addiction: 51}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 85 {
		t.Errorf("expected priority 85, got %d", action.Priority)
	}
	if action.Category != domain.CategoryRehab {
		t.Errorf("expected category %q, got %q", domain.CategoryRehab, action.Category)
	}
}

func TestRehabRule_AtThreshold(t *testing.T) {
	rule := RehabRule{}
	state := domain.PlayerState{Addiction: 50}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil at threshold, got %+v", action)
	}
}

func TestRehabRule_LowAddiction(t *testing.T) {
	rule := RehabRule{}
	state := domain.PlayerState{Addiction: 10}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

// --- GymRule ---

func TestGymRule_EnergyAndHappy(t *testing.T) {
	rule := GymRule{}
	state := domain.PlayerState{Energy: 100, Happy: 5000}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 80 {
		t.Errorf("expected priority 80, got %d", action.Priority)
	}
	if action.Category != domain.CategoryGym {
		t.Errorf("expected category %q, got %q", domain.CategoryGym, action.Category)
	}
}

func TestGymRule_NoEnergy(t *testing.T) {
	rule := GymRule{}
	state := domain.PlayerState{Energy: 0, Happy: 5000}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil when no energy, got %+v", action)
	}
}

func TestGymRule_LowHappy(t *testing.T) {
	rule := GymRule{}
	state := domain.PlayerState{Energy: 100, Happy: 3000}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil when happy too low, got %+v", action)
	}
}

func TestGymRule_HappyAtThreshold(t *testing.T) {
	rule := GymRule{}
	state := domain.PlayerState{Energy: 100, Happy: 4000}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil at happy threshold boundary, got %+v", action)
	}
}

// --- CrimeRule ---

func TestCrimeRule_NerveFull(t *testing.T) {
	rule := CrimeRule{}
	state := domain.PlayerState{Nerve: 60, NerveMax: 60}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 70 {
		t.Errorf("expected priority 70, got %d", action.Priority)
	}
	if action.Category != domain.CategoryCrime {
		t.Errorf("expected category %q, got %q", domain.CategoryCrime, action.Category)
	}
}

func TestCrimeRule_NerveNotFull(t *testing.T) {
	rule := CrimeRule{}
	state := domain.PlayerState{Nerve: 30, NerveMax: 60}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

func TestCrimeRule_NerveMaxZero(t *testing.T) {
	rule := CrimeRule{}
	state := domain.PlayerState{Nerve: 0, NerveMax: 0}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil when NerveMax is 0, got %+v", action)
	}
}

// --- TravelRule ---

func TestTravelRule_Ready(t *testing.T) {
	rule := TravelRule{}
	state := domain.PlayerState{TravelCooldown: 0}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 60 {
		t.Errorf("expected priority 60, got %d", action.Priority)
	}
	if action.Category != domain.CategoryTravel {
		t.Errorf("expected category %q, got %q", domain.CategoryTravel, action.Category)
	}
}

func TestTravelRule_OnCooldown(t *testing.T) {
	rule := TravelRule{}
	state := domain.PlayerState{TravelCooldown: 600}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

// --- DefaultRules ---

func TestDefaultRules_Count(t *testing.T) {
	rules := DefaultRules()
	if len(rules) != 6 {
		t.Errorf("expected 6 default rules, got %d", len(rules))
	}
}
