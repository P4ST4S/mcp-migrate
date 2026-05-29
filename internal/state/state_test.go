package state

import "testing"

func TestDetectListDrift(t *testing.T) {
	drifts := DetectListDrift([]ListObservation{
		{Probe: "tools-list", Method: "tools/list", Accepted: true, Result: map[string]any{"tools": []any{map[string]any{"name": "a"}}}},
		{Probe: "state-tools-list-repeat", Method: "tools/list", Accepted: true, Result: map[string]any{"tools": []any{map[string]any{"name": "b"}}}},
	})
	if len(drifts) != 1 {
		t.Fatalf("expected one drift, got %#v", drifts)
	}
	if drifts[0].Method != "tools/list" {
		t.Fatalf("unexpected method %q", drifts[0].Method)
	}
}

func TestDetectListDriftIgnoresOrder(t *testing.T) {
	drifts := DetectListDrift([]ListObservation{
		{Probe: "resources-list", Method: "resources/list", Accepted: true, Result: map[string]any{"resources": []any{
			map[string]any{"uri": "b"},
			map[string]any{"uri": "a"},
		}}},
		{Probe: "state-resources-list-repeat", Method: "resources/list", Accepted: true, Result: map[string]any{"resources": []any{
			map[string]any{"uri": "a"},
			map[string]any{"uri": "b"},
		}}},
	})
	if len(drifts) != 0 {
		t.Fatalf("expected no order-only drift, got %#v", drifts)
	}
}

func TestDetectListDriftIgnoresExplicitHandleValues(t *testing.T) {
	drifts := DetectListDrift([]ListObservation{
		{Probe: "tools-list", Method: "tools/list", Accepted: true, Result: map[string]any{"tools": []any{
			map[string]any{"name": "checkout", "stateHandle": "state-1"},
		}}},
		{Probe: "state-tools-list-repeat", Method: "tools/list", Accepted: true, Result: map[string]any{"tools": []any{
			map[string]any{"name": "checkout", "stateHandle": "state-2"},
		}}},
	})
	if len(drifts) != 0 {
		t.Fatalf("expected explicit handles to suppress drift, got %#v", drifts)
	}
}
