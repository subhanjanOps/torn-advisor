package engine

import "testing"

func TestWarRule_Active(t *testing.T) {
	rule := WarRule{}
	state := PlayerState{WarActive: true}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 95 {
		t.Errorf("expected priority 95, got %d", action.Priority)
	}
	if action.Category != CategoryWar {
		t.Errorf("expected category %q, got %q", CategoryWar, action.Category)
	}
}

func TestWarRule_Inactive(t *testing.T) {
	rule := WarRule{}
	state := PlayerState{WarActive: false}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

func TestXanaxRule_Ready(t *testing.T) {
	rule := XanaxRule{}
	state := PlayerState{XanaxCooldown: 0}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 90 {
		t.Errorf("expected priority 90, got %d", action.Priority)
	}
	if action.Category != CategoryDrug {
		t.Errorf("expected category %q, got %q", CategoryDrug, action.Category)
	}
}

func TestXanaxRule_OnCooldown(t *testing.T) {
	rule := XanaxRule{}
	state := PlayerState{XanaxCooldown: 300}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

func TestRehabRule_HighAddiction(t *testing.T) {
	rule := RehabRule{}
	state := PlayerState{Addiction: 51}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 85 {
		t.Errorf("expected priority 85, got %d", action.Priority)
	}
	if action.Category != CategoryRehab {
		t.Errorf("expected category %q, got %q", CategoryRehab, action.Category)
	}
}

func TestRehabRule_AtThreshold(t *testing.T) {
	rule := RehabRule{}
	state := PlayerState{Addiction: 50}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil at threshold, got %+v", action)
	}
}

func TestRehabRule_LowAddiction(t *testing.T) {
	rule := RehabRule{}
	state := PlayerState{Addiction: 10}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

func TestGymRule_EnergyAndHappy(t *testing.T) {
	rule := GymRule{}
	state := PlayerState{Energy: 100, Happy: 5000}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 80 {
		t.Errorf("expected priority 80, got %d", action.Priority)
	}
	if action.Category != CategoryGym {
		t.Errorf("expected category %q, got %q", CategoryGym, action.Category)
	}
}

func TestGymRule_NoEnergy(t *testing.T) {
	rule := GymRule{}
	state := PlayerState{Energy: 0, Happy: 5000}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil when no energy, got %+v", action)
	}
}

func TestGymRule_LowHappy(t *testing.T) {
	rule := GymRule{}
	state := PlayerState{Energy: 100, Happy: 3000}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil when happy too low, got %+v", action)
	}
}

func TestGymRule_HappyAtThreshold(t *testing.T) {
	rule := GymRule{}
	state := PlayerState{Energy: 100, Happy: 4000}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil at happy threshold boundary, got %+v", action)
	}
}

func TestCrimeRule_NerveFull(t *testing.T) {
	rule := CrimeRule{}
	state := PlayerState{Nerve: 60, NerveMax: 60}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 70 {
		t.Errorf("expected priority 70, got %d", action.Priority)
	}
	if action.Category != CategoryCrime {
		t.Errorf("expected category %q, got %q", CategoryCrime, action.Category)
	}
}

func TestCrimeRule_NerveNotFull(t *testing.T) {
	rule := CrimeRule{}
	state := PlayerState{Nerve: 30, NerveMax: 60}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

func TestCrimeRule_NerveMaxZero(t *testing.T) {
	rule := CrimeRule{}
	state := PlayerState{Nerve: 0, NerveMax: 0}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil when NerveMax is 0, got %+v", action)
	}
}

func TestTravelRule_Ready(t *testing.T) {
	rule := TravelRule{}
	state := PlayerState{TravelCooldown: 0}
	action := rule.Evaluate(state)

	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Priority != 60 {
		t.Errorf("expected priority 60, got %d", action.Priority)
	}
	if action.Category != CategoryTravel {
		t.Errorf("expected category %q, got %q", CategoryTravel, action.Category)
	}
}

func TestTravelRule_OnCooldown(t *testing.T) {
	rule := TravelRule{}
	state := PlayerState{TravelCooldown: 600}
	if action := rule.Evaluate(state); action != nil {
		t.Errorf("expected nil, got %+v", action)
	}
}

func TestDefaultRules_Count(t *testing.T) {
	rules := DefaultRules()
	if len(rules) != 6 {
		t.Errorf("expected 6 default rules, got %d", len(rules))
	}
}
