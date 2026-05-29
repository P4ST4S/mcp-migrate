package rules

import (
	"fmt"

	"github.com/P4ST4S/mcp-migrate/internal/report"
	"github.com/P4ST4S/mcp-migrate/internal/spec"
)

const (
	StatusConfirmed           = "confirmed"
	StatusPendingVerification = "pending-verification"
)

type SEP struct {
	ID        string
	Status    string
	Source    string
	FileFound bool
}

type Rule struct {
	ID                                string
	SEP                               SEP
	Severity                          report.Severity
	AppliesTo                         []string
	Autofixable                       bool
	Status                            string
	EnforceUnverifiedSEPJustification string
	Message                           string
	Remediation                       string
}

func (r Rule) Enforcement() report.Enforcement {
	if r.Status == StatusPendingVerification {
		return report.EnforcementReportOnly
	}
	sep := r.SEPRef()
	if sep != nil && sep.Verification == report.SEPUnverified && r.EnforceUnverifiedSEPJustification == "" {
		return report.EnforcementReportOnly
	}
	return report.EnforcementEnforced
}

func (r Rule) SEPRef() *report.SEPRef {
	if r.SEP.ID == "" {
		return nil
	}
	verification := report.SEPUnverified
	if r.SEP.Status == "Final" && r.SEP.FileFound {
		verification = report.SEPVerified
	}
	return &report.SEPRef{
		ID:           r.SEP.ID,
		Status:       r.SEP.Status,
		Verification: verification,
		Source:       r.SEP.Source,
	}
}

func (r Rule) Finding(source report.Source) report.Finding {
	return report.Finding{
		Schema:      spec.FindingSchema,
		Rule:        r.ID,
		SEP:         r.SEPRef(),
		Severity:    r.Severity,
		Enforcement: r.Enforcement(),
		SpecTarget:  spec.TargetVersion,
		Source:      source,
		Message:     r.Message,
		Remediation: r.Remediation,
		Autofix:     r.Autofixable,
		Status:      r.Status,
	}
}

type Registry struct {
	rules []Rule
}

func NewRegistry(rules []Rule) (*Registry, error) {
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		if err := validateRule(rule); err != nil {
			return nil, err
		}
		if _, ok := seen[rule.ID]; ok {
			return nil, fmt.Errorf("duplicate rule id %q", rule.ID)
		}
		seen[rule.ID] = struct{}{}
	}
	return &Registry{rules: append([]Rule(nil), rules...)}, nil
}

func (r *Registry) All() []Rule {
	if r == nil {
		return nil
	}
	return append([]Rule(nil), r.rules...)
}

func (r *Registry) Find(id string) (Rule, bool) {
	if r == nil {
		return Rule{}, false
	}
	for _, rule := range r.rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return Rule{}, false
}

func DefaultRegistry() (*Registry, error) {
	return NewRegistry(defaultRules)
}

func validateRule(rule Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule id is required")
	}
	if rule.SEP.ID == "" {
		return fmt.Errorf("rule %q has no SEP reference", rule.ID)
	}
	switch rule.Severity {
	case report.SeverityBreaking, report.SeverityDeprecated, report.SeverityWarning, report.SeverityInfo:
	default:
		return fmt.Errorf("rule %q has unknown severity %q", rule.ID, rule.Severity)
	}
	switch rule.Status {
	case StatusConfirmed, StatusPendingVerification:
	default:
		return fmt.Errorf("rule %q has unknown status %q", rule.ID, rule.Status)
	}
	return nil
}

func sep(id, status, source string, fileFound bool) SEP {
	return SEP{ID: id, Status: status, Source: source, FileFound: fileFound}
}

var defaultRules = []Rule{
	{
		ID:          "initialize-handshake-removed",
		SEP:         sep("SEP-2575", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2575-stateless-mcp.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Legacy initialize handshake is not the MCP 2026-07-28 stateless path.",
		Remediation: "Carry protocol version, client info, and client capabilities in per-request _meta and expose server/discover.",
	},
	{
		ID:          "initialize-text-heuristic",
		SEP:         sep("SEP-2575", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2575-stateless-mcp.md", true),
		Severity:    report.SeverityWarning,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Response text mentions initialize, which is only a weak legacy signal.",
		Remediation: "Confirm with stronger evidence such as server/discover failure plus a successful legacy initialize probe on an isolated stdio process.",
	},
	{
		ID:          "server-discover-required",
		SEP:         sep("SEP-2575", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2575-stateless-mcp.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Server does not expose the stateless server/discover RPC.",
		Remediation: "Implement server/discover with supported versions, server capabilities, and server identity.",
	},
	{
		ID:          "protocol-version-per-request",
		SEP:         sep("SEP-2575", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2575-stateless-mcp.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Request is missing per-request protocol version metadata or HTTP version header.",
		Remediation: "Send io.modelcontextprotocol/protocolVersion in _meta on every request and MCP-Protocol-Version on HTTP.",
	},
	{
		ID:          "client-info-capabilities-per-request",
		SEP:         sep("SEP-2575", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2575-stateless-mcp.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Request is missing per-request client identity or capabilities.",
		Remediation: "Send io.modelcontextprotocol/clientInfo and io.modelcontextprotocol/clientCapabilities in request _meta.",
	},
	{
		ID:          "mcp-session-id-removed",
		SEP:         sep("SEP-2567", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2567-sessionless-mcp.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Server depends on the removed Mcp-Session-Id protocol session header.",
		Remediation: "Replace protocol sessions with explicit application handles passed through tool arguments and results.",
	},
	{
		ID:          "session-dependent-lists-removed",
		SEP:         sep("SEP-2567", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2567-sessionless-mcp.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "List results appear to vary by connection or hidden session state.",
		Remediation: "Make tools/list, resources/list, and prompts/list independent of connection/session state.",
	},
	{
		ID:          "explicit-state-handles",
		SEP:         sep("SEP-2567", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2567-sessionless-mcp.md", true),
		Severity:    report.SeverityWarning,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Stateful workflow is not represented by explicit handles.",
		Remediation: "Mint opaque handles in tool results and require them as ordinary arguments on subsequent calls.",
	},
	{
		ID:          "http-standard-headers",
		SEP:         sep("SEP-2243", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2243-http-standardization.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Streamable HTTP request is missing required MCP routing headers or accepts header/body mismatch.",
		Remediation: "Send and validate Mcp-Method on all POSTs and Mcp-Name for tools/call, resources/read, and prompts/get.",
	},
	{
		ID:          "x-mcp-header",
		SEP:         sep("SEP-2243", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2243-http-standardization.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusConfirmed,
		Message:     "Tool parameter header mirroring via x-mcp-header is not supported or has invalid declarations.",
		Remediation: "Mirror valid annotated primitive tool parameters into Mcp-Param-* headers and reject invalid tool definitions for HTTP clients.",
	},
	{
		ID:          "mrtr-input-required",
		SEP:         sep("SEP-2322", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2322-MRTR.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusPendingVerification,
		Message:     "Server uses a legacy server-initiated request flow instead of MRTR input-required results.",
		Remediation: "Use inputRequests, inputResponses, requestState, and the final spec resultType spelling once reconciled.",
	},
	{
		ID:          "request-scoped-server-requests-only",
		SEP:         sep("SEP-2260", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2260-Require-Server-requests-to-be-associated-with-Client-requests.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Server sends standalone server-to-client requests outside an originating client request.",
		Remediation: "Associate roots/list, sampling/createMessage, and elicitation/create with an originating request or migrate to MRTR.",
	},
	{
		ID:          "subscriptions-listen",
		SEP:         sep("SEP-2575", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2575-stateless-mcp.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Server exposes legacy GET SSE or resource subscription methods instead of subscriptions/listen.",
		Remediation: "Use subscriptions/listen over POST with explicit notification opt-ins.",
	},
	{
		ID:          "ping-removed",
		SEP:         sep("SEP-2575", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2575-stateless-mcp.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Server relies on the removed ping RPC.",
		Remediation: "Use ordinary RPC calls and transport-level health checks instead of MCP ping.",
	},
	{
		ID:          "logging-setlevel-replaced",
		SEP:         sep("SEP-2575", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2575-stateless-mcp.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusPendingVerification,
		Message:     "Server relies on logging/setLevel instead of per-request logLevel metadata.",
		Remediation: "Re-check final spec, then use io.modelcontextprotocol/logLevel in per-request _meta if the removal remains.",
	},
	{
		ID:          "roots-deprecated",
		SEP:         sep("SEP-2577", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2577-deprecate-roots-sampling-and-logging.md", true),
		Severity:    report.SeverityDeprecated,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusConfirmed,
		Message:     "Roots is deprecated but remains functional during the lifecycle window.",
		Remediation: "Pass directories/files through tool parameters, resource URIs, or server configuration.",
	},
	{
		ID:          "sampling-deprecated",
		SEP:         sep("SEP-2577", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2577-deprecate-roots-sampling-and-logging.md", true),
		Severity:    report.SeverityDeprecated,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusConfirmed,
		Message:     "Sampling is deprecated but remains functional during the lifecycle window.",
		Remediation: "Integrate directly with LLM provider APIs for new implementations.",
	},
	{
		ID:          "logging-deprecated",
		SEP:         sep("SEP-2577", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2577-deprecate-roots-sampling-and-logging.md", true),
		Severity:    report.SeverityDeprecated,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusPendingVerification,
		Message:     "Logging is deprecated, with method-level details pending final spec reconciliation.",
		Remediation: "Use stderr for stdio and OpenTelemetry for structured observability.",
	},
	{
		ID:          "http-sse-transport-deprecated",
		SEP:         sep("SEP-2596", "Draft", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2596-spec-feature-lifecycle-and-deprecation.md", true),
		Severity:    report.SeverityDeprecated,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusConfirmed,
		Message:     "HTTP+SSE transport is in the Deprecated lifecycle state.",
		Remediation: "Migrate to Streamable HTTP.",
	},
	{
		ID:          "includecontext-values-deprecated",
		SEP:         sep("SEP-2596", "Draft", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2596-spec-feature-lifecycle-and-deprecation.md", true),
		Severity:    report.SeverityDeprecated,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusConfirmed,
		Message:     "Sampling includeContext values thisServer/allServers are deprecated.",
		Remediation: "Omit includeContext or use none.",
	},
	{
		ID:          "tasks-core-to-extension",
		SEP:         sep("SEP-2663", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2663-tasks-extension.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusConfirmed,
		Message:     "Server uses the experimental core tasks surface instead of the official tasks extension.",
		Remediation: "Negotiate io.modelcontextprotocol/tasks and use tasks/get, tasks/update, and tasks/cancel.",
	},
	{
		ID:          "extensions-capabilities",
		SEP:         sep("SEP-2133", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2133-extensions.md", true),
		Severity:    report.SeverityInfo,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusConfirmed,
		Message:     "Implementation could advertise optional extension capabilities more explicitly.",
		Remediation: "Use ClientCapabilities.extensions and ServerCapabilities.extensions with reverse-DNS extension identifiers.",
	},
	{
		ID:          "cacheable-results-required",
		SEP:         sep("SEP-2549", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2549-TTL-for-list-results.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live"},
		Status:      StatusConfirmed,
		Message:     "Cacheable result is missing ttlMs or cacheScope.",
		Remediation: "Return ttlMs and cacheScope on list/read results covered by the 2026-07-28 draft caching spec; this remains report-only until SEP-2549 is Final.",
	},
	{
		ID:          "trace-context-meta",
		SEP:         sep("SEP-414", "Final", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/414-request-meta.md", true),
		Severity:    report.SeverityInfo,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusConfirmed,
		Message:     "Trace context propagation can use standard _meta keys.",
		Remediation: "Use traceparent, tracestate, and baggage in _meta for W3C trace context propagation.",
	},
	{
		ID:          "resource-not-found-code",
		SEP:         sep("SEP-2164", "Draft", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2164-resource-not-found-error.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live", "static"},
		Autofixable: true,
		Status:      StatusPendingVerification,
		Message:     "Resource not found uses a legacy or non-standard JSON-RPC error code.",
		Remediation: "Use -32602 Invalid Params for missing resources after final spec reconciliation.",
	},
	{
		ID:          "json-schema-2020-12-tools",
		SEP:         sep("SEP-2106", "Draft", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2106-json-schema-2020-12.md", true),
		Severity:    report.SeverityWarning,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusPendingVerification,
		Message:     "Tool schema validation may reject JSON Schema 2020-12 constructs expected by the draft.",
		Remediation: "Allow JSON Schema 2020-12 keywords in tool schemas once the final spec confirms the shape.",
	},
	{
		ID:          "auth-iss-validation",
		SEP:         sep("SEP-2468", "In-Review", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2468-recommend-issuer-claim-for-auth.md", true),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"live", "static"},
		Status:      StatusPendingVerification,
		Message:     "OAuth authorization response issuer validation is missing or incomplete.",
		Remediation: "Record the expected issuer from validated metadata and compare authorization response iss using simple string comparison.",
	},
	{
		ID:          "auth-application-type-dcr",
		SEP:         sep("SEP-837", "unindexed", "", false),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"static"},
		Status:      StatusPendingVerification,
		Message:     "Dynamic Client Registration omits application_type.",
		Remediation: "Set application_type to native for CLI/localhost/native clients or web for remote browser apps.",
	},
	{
		ID:          "auth-server-binding",
		SEP:         sep("SEP-2352", "unindexed", "", false),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"static"},
		Status:      StatusPendingVerification,
		Message:     "Persisted OAuth client credentials may not be bound to the issuing authorization server.",
		Remediation: "Key persisted credentials by authorization-server issuer and re-register when the issuer changes.",
	},
	{
		ID:          "auth-step-up-scopes",
		SEP:         sep("SEP-2350", "unindexed", "", false),
		Severity:    report.SeverityWarning,
		AppliesTo:   []string{"static"},
		Status:      StatusPendingVerification,
		Message:     "Step-up authorization may lose existing scopes or ignore challenge scopes.",
		Remediation: "Treat challenged scopes as authoritative for the current operation and preserve previously granted scopes during re-authorization.",
	},
	{
		ID:          "auth-well-known-suffix",
		SEP:         sep("SEP-2351", "unindexed", "", false),
		Severity:    report.SeverityBreaking,
		AppliesTo:   []string{"static"},
		Status:      StatusPendingVerification,
		Message:     "Authorization-server discovery may use obsolete or unvalidated well-known metadata behavior.",
		Remediation: "Use RFC 8414/OIDC discovery order and reject metadata whose issuer does not exactly match the expected issuer.",
	},
	{
		ID:          "auth-refresh-token-guidance",
		SEP:         sep("SEP-2207", "Accepted", "https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/seps/2207-oidc-refresh-token-guidance.md", true),
		Severity:    report.SeverityInfo,
		AppliesTo:   []string{"static"},
		Status:      StatusPendingVerification,
		Message:     "Refresh-token handling should follow MCP auth guidance.",
		Remediation: "Request refresh tokens explicitly when desired and do not advertise offline_access as a protected-resource requirement.",
	},
}
