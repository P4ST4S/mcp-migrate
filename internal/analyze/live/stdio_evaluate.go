package live

import (
	"fmt"

	"github.com/P4ST4S/mcp-migrate/internal/report"
	"github.com/P4ST4S/mcp-migrate/internal/rules"
	"github.com/P4ST4S/mcp-migrate/internal/state"
)

func EvaluateSTDIOTrace(trace STDIOTrace, registry *rules.Registry) []report.Finding {
	source := report.Source{Mode: "live", Ref: trace.Command}
	builder := findingBuilder{registry: registry, source: source}

	byProbe := make(map[string]STDIOObservation, len(trace.Observations))
	for _, obs := range trace.Observations {
		byProbe[obs.Probe] = obs
	}

	discover, hasDiscover := byProbe["discover"]
	if !hasDiscover || !discover.Accepted() || !hasDiscoverShape(discover.Result) {
		builder.add("server-discover-required", describeSTDIOObservation(discover))
	}

	if initObs, ok := byProbe["initialize-legacy"]; ok && initObs.Accepted() {
		builder.add("initialize-handshake-removed", "Legacy initialize succeeded on an isolated stdio process after server/discover failed.")
	}

	if obs, ok := byProbe["tools-list-missing-meta"]; ok && obs.Accepted() {
		builder.add("client-info-capabilities-per-request", "Server accepted read-only stdio tools/list with no per-request _meta.")
	}

	for _, name := range []string{"tools-list", "resources-list", "prompts-list"} {
		obs, ok := byProbe[name]
		if !ok || !obs.Accepted() {
			continue
		}
		if missingCacheableFields(obs.Result) {
			builder.add("cacheable-results-required", fmt.Sprintf("%s stdio response was accepted but did not include both ttlMs and cacheScope.", obs.RPCMethod))
		}
	}

	for _, drift := range state.DetectListDrift(stdioListObservations(trace)) {
		builder.add("explicit-state-handles", fmt.Sprintf("%s stdio result changed within one analyzer-owned process between read-only probes %s and %s.", drift.Method, drift.FirstProbe, drift.SecondProbe))
	}

	return builder.findings
}

func stdioListObservations(trace STDIOTrace) []state.ListObservation {
	observations := make([]state.ListObservation, 0, len(trace.Observations))
	for _, obs := range trace.Observations {
		if !isListMethod(obs.RPCMethod) {
			continue
		}
		observations = append(observations, state.ListObservation{
			Probe:    obs.Probe,
			Method:   obs.RPCMethod,
			Accepted: obs.Accepted(),
			Result:   obs.Result,
		})
	}
	return observations
}

func describeSTDIOObservation(obs STDIOObservation) string {
	if obs.Timeout {
		return "Probe timed out; process was cancelled and killed if still running."
	}
	if obs.ProcessError {
		return "Probe failed due to a process error. Command args, environment, and stderr are redacted."
	}
	if obs.ParseError {
		return fmt.Sprintf("Probe %s did not return a parseable JSON-RPC response.", obs.Probe)
	}
	if obs.HasRPCError {
		return fmt.Sprintf("Probe %s returned JSON-RPC error code %d.", obs.Probe, obs.RPCErrorCode)
	}
	return "Probe did not produce a usable response."
}
