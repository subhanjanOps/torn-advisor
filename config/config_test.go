package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPriorities(t *testing.T) {
	p := DefaultPriorities()

	expected := map[string]int{
		"Hospital": 98, "Chain": 97, "War": 95,
		"Xanax": 90, "Rehab": 85, "Gym": 80,
		"Crime": 70, "Travel": 60, "Booster": 55,
	}

	actual := map[string]int{
		"Hospital": p.Hospital, "Chain": p.Chain, "War": p.War,
		"Xanax": p.Xanax, "Rehab": p.Rehab, "Gym": p.Gym,
		"Crime": p.Crime, "Travel": p.Travel, "Booster": p.Booster,
	}

	for name, want := range expected {
		if got := actual[name]; got != want {
			t.Errorf("%s: expected %d, got %d", name, want, got)
		}
	}
}

func TestLoadPriorities_FullOverride(t *testing.T) {
	content := `{"hospital":10,"chain":20,"war":30,"xanax":40,"rehab":50,"gym":60,"crime":70,"travel":80,"booster":90}`
	path := writeTemp(t, content)

	p, err := LoadPriorities(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Hospital != 10 || p.Booster != 90 || p.Gym != 60 {
		t.Errorf("unexpected priorities: %+v", p)
	}
}

func TestLoadPriorities_PartialOverride(t *testing.T) {
	content := `{"gym":99}`
	path := writeTemp(t, content)

	p, err := LoadPriorities(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Gym != 99 {
		t.Errorf("expected gym=99, got %d", p.Gym)
	}
	// Unspecified fields retain defaults.
	if p.Hospital != 98 {
		t.Errorf("expected hospital=98 (default), got %d", p.Hospital)
	}
}

func TestLoadPriorities_FileNotFound(t *testing.T) {
	_, err := LoadPriorities("/nonexistent/path.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadPriorities_InvalidJSON(t *testing.T) {
	path := writeTemp(t, `{bad json}`)
	_, err := LoadPriorities(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadPriorities_EmptyJSON(t *testing.T) {
	path := writeTemp(t, `{}`)
	p, err := LoadPriorities(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All defaults should be preserved.
	d := DefaultPriorities()
	if p != d {
		t.Errorf("expected defaults %+v, got %+v", d, p)
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "priorities.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}
