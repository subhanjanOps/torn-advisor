package engine

import (
	"testing"

	"github.com/subhanjanOps/torn-advisor/domain"
)

func TestBuildPlan_FiltersNils(t *testing.T) {
	actions := []*domain.Action{
		nil,
		{Name: "A", Priority: 50},
		nil,
		{Name: "B", Priority: 80},
		nil,
	}
	plan := BuildPlan(actions)

	if len(plan) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(plan))
	}
}

func TestBuildPlan_SortsByPriorityDesc(t *testing.T) {
	actions := []*domain.Action{
		{Name: "Low", Priority: 30},
		{Name: "High", Priority: 90},
		{Name: "Mid", Priority: 60},
	}
	plan := BuildPlan(actions)

	if len(plan) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(plan))
	}
	if plan[0].Name != "High" {
		t.Errorf("expected first action 'High', got %q", plan[0].Name)
	}
	if plan[1].Name != "Mid" {
		t.Errorf("expected second action 'Mid', got %q", plan[1].Name)
	}
	if plan[2].Name != "Low" {
		t.Errorf("expected third action 'Low', got %q", plan[2].Name)
	}
}

func TestBuildPlan_Empty(t *testing.T) {
	plan := BuildPlan(nil)
	if len(plan) != 0 {
		t.Errorf("expected empty plan, got %d actions", len(plan))
	}
}

func TestBuildPlan_AllNils(t *testing.T) {
	actions := []*domain.Action{nil, nil, nil}
	plan := BuildPlan(actions)
	if len(plan) != 0 {
		t.Errorf("expected empty plan from all nils, got %d actions", len(plan))
	}
}

func TestBuildPlan_EqualPriority(t *testing.T) {
	actions := []*domain.Action{
		{Name: "A", Priority: 70},
		{Name: "B", Priority: 70},
	}
	plan := BuildPlan(actions)

	if len(plan) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(plan))
	}
	// Both present; stable order not guaranteed, but both must exist.
	names := map[string]bool{plan[0].Name: true, plan[1].Name: true}
	if !names["A"] || !names["B"] {
		t.Errorf("expected both A and B, got %q and %q", plan[0].Name, plan[1].Name)
	}
}
