package rules

import "testing"

func TestNewRegistryRejectsDuplicateIDs(t *testing.T) {
	_, err := NewRegistry([]Rule{
		{ID: "same", Severity: SeverityWarning},
		{ID: "same", Severity: SeverityInfo},
	})
	if err == nil {
		t.Fatal("expected duplicate rule error")
	}
}

func TestDefaultRegistryIsEmptyForScaffold(t *testing.T) {
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry returned error: %v", err)
	}
	if got := len(registry.All()); got != 0 {
		t.Fatalf("expected no active rules in scaffold, got %d", got)
	}
}
