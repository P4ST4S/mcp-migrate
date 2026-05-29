package live

import (
	"testing"

	"github.com/P4ST4S/mcp-migrate/internal/report"
)

func TestEvaluateHTTPTraceDetectsListDrift(t *testing.T) {
	registry := mustRegistry(t)
	findings := EvaluateHTTPTrace(HTTPTrace{
		Endpoint: "http://example.test/mcp",
		Observations: []HTTPObservation{
			{Probe: "tools-list", RPCMethod: "tools/list", StatusCode: 200, HasResult: true, Result: map[string]any{"tools": []any{map[string]any{"name": "tool-1"}}}},
			{Probe: "state-tools-list-repeat", RPCMethod: "tools/list", StatusCode: 200, HasResult: true, Result: map[string]any{"tools": []any{map[string]any{"name": "tool-2"}}}},
		},
	}, registry)

	assertFinding(t, findings, "session-dependent-lists-removed")
	assertFindingEnforcement(t, findings, "session-dependent-lists-removed", report.EnforcementEnforced)
}

func TestEvaluateHTTPTraceIgnoresExplicitHandleOnlyDrift(t *testing.T) {
	registry := mustRegistry(t)
	findings := EvaluateHTTPTrace(HTTPTrace{
		Endpoint: "http://example.test/mcp",
		Observations: []HTTPObservation{
			{Probe: "tools-list", RPCMethod: "tools/list", StatusCode: 200, HasResult: true, Result: map[string]any{"tools": []any{map[string]any{"name": "checkout", "stateHandle": "state-1"}}}},
			{Probe: "state-tools-list-repeat", RPCMethod: "tools/list", StatusCode: 200, HasResult: true, Result: map[string]any{"tools": []any{map[string]any{"name": "checkout", "stateHandle": "state-2"}}}},
		},
	}, registry)

	assertNoFinding(t, findings, "session-dependent-lists-removed")
	assertNoFinding(t, findings, "explicit-state-handles")
}

func TestEvaluateSTDIOTraceDetectsProcessLifetimeListDrift(t *testing.T) {
	registry := mustRegistry(t)
	findings := EvaluateSTDIOTrace(STDIOTrace{
		Command: "fixture",
		Observations: []STDIOObservation{
			{Probe: "tools-list", RPCMethod: "tools/list", HasResult: true, Result: map[string]any{"tools": []any{map[string]any{"name": "tool-1"}}}},
			{Probe: "state-tools-list-repeat", RPCMethod: "tools/list", HasResult: true, Result: map[string]any{"tools": []any{map[string]any{"name": "tool-2"}}}},
		},
	}, registry)

	assertFinding(t, findings, "explicit-state-handles")
	assertFindingSeverity(t, findings, "explicit-state-handles", report.SeverityWarning)
	assertNoFinding(t, findings, "session-dependent-lists-removed")
}
