package engine

import (
	"testing"

	"github.com/subhanjanOps/torn-advisor/domain"
	"github.com/subhanjanOps/torn-advisor/rules"
)

func TestNewEngine(t *testing.T) {
	eng := NewEngine(rules.DefaultRules())
	if eng == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestEngine_Run_AllRulesFire(t *testing.T) {
	eng := NewEngine(rules.DefaultRules())
	state := domain.PlayerState{
		Life:            1000,
		LifeMax:         7500,
		ChainActive:     true,
		WarActive:       true,
		XanaxCooldown:   0,
		Addiction:       60,
		Energy:          100,
		Happy:           5000,
		Nerve:           60,
		NerveMax:        60,
		TravelCooldown:  0,
		BoosterCooldown: 0,
	}

	plan := eng.Run(state)
	if len(plan) != 9 {
		t.Fatalf("expected 9 actions, got %d", len(plan))
	}

	// Verify priority ordering: 98, 97, 95, 90, 85, 80, 70, 60, 55
	expectedPriorities := []int{98, 97, 95, 90, 85, 80, 70, 60, 55}
	for i, want := range expectedPriorities {
		if plan[i].Priority != want {
			t.Errorf("action[%d]: expected priority %d, got %d (%s)", i, want, plan[i].Priority, plan[i].Name)
		}
	}
}

func TestEngine_Run_NoRulesFire(t *testing.T) {
	eng := NewEngine(rules.DefaultRules())
	state := domain.PlayerState{
		WarActive:       false,
		XanaxCooldown:   300,
		Addiction:       10,
		Energy:          0,
		Happy:           1000,
		Nerve:           0,
		NerveMax:        60,
		TravelCooldown:  600,
		BoosterCooldown: 120,
		Life:            7500,
		LifeMax:         7500,
		ChainActive:     false,
	}

	plan := eng.Run(state)
	if len(plan) != 0 {
		t.Errorf("expected empty plan, got %d actions", len(plan))
	}
}

func TestEngine_Run_PartialRulesFire(t *testing.T) {
	eng := NewEngine(rules.DefaultRules())
	state := domain.PlayerState{
		WarActive:       false,
		XanaxCooldown:   300,
		Addiction:       60, // rehab fires
		Energy:          100,
		Happy:           5000, // gym fires
		Nerve:           30,
		NerveMax:        60,  // crime does NOT fire
		TravelCooldown:  600, // travel does NOT fire
		BoosterCooldown: 120, // booster does NOT fire
		Life:            7500,
		LifeMax:         7500,  // hospital does NOT fire
		ChainActive:     false, // chain does NOT fire
	}

	plan := eng.Run(state)
	if len(plan) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(plan))
	}

	if plan[0].Category != domain.CategoryRehab {
		t.Errorf("expected first action to be rehab, got %q", plan[0].Category)
	}
	if plan[1].Category != domain.CategoryGym {
		t.Errorf("expected second action to be gym, got %q", plan[1].Category)
	}
}

func TestEngine_Run_NoRules(t *testing.T) {
	eng := NewEngine(nil)
	plan := eng.Run(domain.PlayerState{})
	if len(plan) != 0 {
		t.Errorf("expected empty plan with no rules, got %d actions", len(plan))
	}
}

// stubRule is a test helper that always returns a fixed action.
type stubRule struct {
	action *domain.Action
}

func (s stubRule) Evaluate(_ domain.PlayerState) *domain.Action {
	return s.action
}

func TestEngine_Run_CustomRules(t *testing.T) {
	custom := stubRule{action: &domain.Action{
		Name:     "Custom",
		Priority: 99,
		Category: "custom",
	}}

	eng := NewEngine([]Rule{custom})
	plan := eng.Run(domain.PlayerState{})

	if len(plan) != 1 {
		t.Fatalf("expected 1 action, got %d", len(plan))
	}
	if plan[0].Name != "Custom" {
		t.Errorf("expected 'Custom', got %q", plan[0].Name)
	}
}
