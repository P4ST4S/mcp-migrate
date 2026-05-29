package live

import (
	"fmt"
	"strings"

	"github.com/P4ST4S/mcp-migrate/internal/report"
	"github.com/P4ST4S/mcp-migrate/internal/rules"
)

func EvaluateHTTPTrace(trace HTTPTrace, registry *rules.Registry) []report.Finding {
	source := report.Source{Mode: "live", Ref: trace.Endpoint}
	builder := findingBuilder{registry: registry, source: source}

	byProbe := make(map[string]HTTPObservation, len(trace.Observations))
	for _, obs := range trace.Observations {
		byProbe[obs.Probe] = obs
		if obs.HasMcpSessionID || obs.BodyMentionsSessionID {
			builder.add("mcp-session-id-removed", "A response indicated Mcp-Session-Id usage. Header values and body content are redacted.")
		}
		if obs.BodyMentionsInitialize {
			builder.add("initialize-text-heuristic", "A response mentioned initialize. This is a weak heuristic only; body content is redacted.")
		}
	}

	discover, hasDiscover := byProbe["discover"]
	if !hasDiscover || !discover.Accepted() || !hasDiscoverShape(discover.Result) {
		builder.add("server-discover-required", describeObservation(discover))
	}

	if obs, ok := byProbe["discover-version-mismatch"]; ok && obs.Accepted() {
		builder.add("protocol-version-per-request", "Server accepted a read-only server/discover probe where MCP-Protocol-Version did not match request _meta.")
	}
	if obs, ok := byProbe["tools-list-missing-meta"]; ok && obs.Accepted() {
		builder.add("client-info-capabilities-per-request", "Server accepted a read-only tools/list probe with no per-request _meta.")
	}

	if obs, ok := byProbe["tools-list-missing-method-header"]; ok && obs.Accepted() {
		builder.add("http-standard-headers", "Server accepted read-only tools/list without Mcp-Method.")
	}
	if obs, ok := byProbe["tools-list-mismatched-method-header"]; ok && obs.Accepted() {
		builder.add("http-standard-headers", "Server accepted read-only tools/list with Mcp-Method that did not match the JSON-RPC method.")
	}

	cacheableProbes := []string{"tools-list", "resources-list", "resources-read", "prompts-list"}
	for _, name := range cacheableProbes {
		obs, ok := byProbe[name]
		if !ok || !obs.Accepted() {
			continue
		}
		if missingCacheableFields(obs.Result) {
			builder.add("cacheable-results-required", fmt.Sprintf("%s response was accepted but did not include both ttlMs and cacheScope.", obs.RPCMethod))
		}
	}

	if obs, ok := byProbe["resources-read"]; ok && obs.HasRPCError && obs.RPCErrorCode == -32002 {
		builder.add("resource-not-found-code", "resources/read returned legacy JSON-RPC error code -32002 for a missing resource.")
	}

	return builder.findings
}

type findingBuilder struct {
	registry *rules.Registry
	source   report.Source
	seen     map[string]struct{}
	findings []report.Finding
}

func (b *findingBuilder) add(ruleID, detail string) {
	if b.seen == nil {
		b.seen = make(map[string]struct{})
	}
	key := ruleID + "\x00" + detail
	if _, ok := b.seen[key]; ok {
		return
	}
	b.seen[key] = struct{}{}

	rule, ok := b.registry.Find(ruleID)
	if !ok {
		return
	}
	finding := rule.Finding(b.source)
	finding.Detail = redactDetail(detail)
	b.findings = append(b.findings, finding)
}

func hasDiscoverShape(result map[string]any) bool {
	if result == nil {
		return false
	}
	if _, ok := result["supportedVersions"]; !ok {
		return false
	}
	if _, ok := result["capabilities"]; !ok {
		return false
	}
	if _, ok := result["serverInfo"]; !ok {
		return false
	}
	return true
}

func missingCacheableFields(result map[string]any) bool {
	if result == nil {
		return true
	}
	if _, ok := result["ttlMs"]; !ok {
		return true
	}
	if _, ok := result["cacheScope"]; !ok {
		return true
	}
	return false
}

func describeObservation(obs HTTPObservation) string {
	if obs.NetworkError {
		return "Probe failed due to a network error. Error details are redacted."
	}
	if obs.ParseError {
		return fmt.Sprintf("Probe %s returned HTTP %d but not a JSON-RPC response.", obs.Probe, obs.StatusCode)
	}
	if obs.HasRPCError {
		return fmt.Sprintf("Probe %s returned JSON-RPC error code %d.", obs.Probe, obs.RPCErrorCode)
	}
	if obs.StatusCode != 0 {
		return fmt.Sprintf("Probe %s returned HTTP %d.", obs.Probe, obs.StatusCode)
	}
	return "Probe did not produce a usable response."
}

func redactDetail(detail string) string {
	parts := strings.Fields(detail)
	for i, part := range parts {
		if !strings.Contains(part, "=") {
			continue
		}
		key := strings.Trim(strings.Split(part, "=")[0], ":,.;()[]{}\"'")
		if isSensitiveDetailName(key) {
			parts[i] = key + "=redacted"
		}
	}
	return strings.Join(parts, " ")
}

func isSensitiveDetailName(name string) bool {
	normalized := strings.ToLower(name)
	for _, part := range []string{
		"authorization",
		"token",
		"secret",
		"password",
		"passwd",
		"api-key",
		"apikey",
		"credential",
		"cookie",
		"session",
	} {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}
