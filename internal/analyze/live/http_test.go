package live

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/P4ST4S/mcp-migrate/internal/report"
	"github.com/P4ST4S/mcp-migrate/internal/spec"
)

func TestHTTPAnalyzeCompliantServer(t *testing.T) {
	fixture := newHTTPFixture(t, compliantProfile)
	defer fixture.Close()

	findings, err := Analyze(Options{Transport: "http", URL: fixture.URL, SpecTarget: spec.TargetVersion})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
	assertNoToolCall(t, fixture.Methods())
}

func TestHTTPAnalyzeLegacyServer(t *testing.T) {
	fixture := newHTTPFixture(t, legacyProfile)
	defer fixture.Close()

	findings, err := Analyze(Options{Transport: "http", URL: fixture.URL + "?access_token=super-secret", SpecTarget: spec.TargetVersion})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	assertFinding(t, findings, "server-discover-required")
	assertFinding(t, findings, "mcp-session-id-removed")
	assertFinding(t, findings, "initialize-text-heuristic")
	assertFindingEnforcement(t, findings, "server-discover-required", report.EnforcementReportOnly)
	assertFindingEnforcement(t, findings, "mcp-session-id-removed", report.EnforcementEnforced)
	assertFindingEnforcement(t, findings, "initialize-text-heuristic", report.EnforcementReportOnly)
	assertNoFinding(t, findings, "initialize-handshake-removed")
	assertNoToolCall(t, fixture.Methods())
	assertNoSecretLeak(t, findings, "super-secret")
	assertNoSecretLeak(t, findings, "legacy-session-secret")
}

func TestHTTPAnalyzeMixedServer(t *testing.T) {
	fixture := newHTTPFixture(t, mixedProfile)
	defer fixture.Close()

	findings, err := Analyze(Options{Transport: "http", URL: fixture.URL, SpecTarget: spec.TargetVersion})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	assertFinding(t, findings, "http-standard-headers")
	assertFinding(t, findings, "cacheable-results-required")
	assertFinding(t, findings, "client-info-capabilities-per-request")
	assertFindingEnforcement(t, findings, "http-standard-headers", report.EnforcementEnforced)
	assertFindingEnforcement(t, findings, "cacheable-results-required", report.EnforcementReportOnly)
	assertFindingEnforcement(t, findings, "client-info-capabilities-per-request", report.EnforcementReportOnly)
	assertNoFinding(t, findings, "mcp-session-id-removed")
	assertNoToolCall(t, fixture.Methods())
}

func TestHTTPAnalyzeDoesNotResourceReadByDefault(t *testing.T) {
	fixture := newHTTPFixture(t, compliantProfile)
	defer fixture.Close()

	_, err := Analyze(Options{Transport: "http", URL: fixture.URL, SpecTarget: spec.TargetVersion})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if slices.Contains(fixture.Methods(), "resources/read") {
		t.Fatalf("resources/read should be opt-in, got methods %v", fixture.Methods())
	}
}

func TestHTTPAnalyzeResourceReadOptIn(t *testing.T) {
	fixture := newHTTPFixture(t, compliantProfile)
	defer fixture.Close()

	_, err := Analyze(Options{Transport: "http", URL: fixture.URL, SpecTarget: spec.TargetVersion, AllowResourceRead: true})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if !slices.Contains(fixture.Methods(), "resources/read") {
		t.Fatalf("expected resources/read with opt-in, got methods %v", fixture.Methods())
	}
}

func TestHTTPResourceNotFoundCodeIsPendingReportOnly(t *testing.T) {
	fixture := newHTTPFixture(t, resourceNotFoundProfile)
	defer fixture.Close()

	findings, err := Analyze(Options{Transport: "http", URL: fixture.URL, SpecTarget: spec.TargetVersion, AllowResourceRead: true})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	assertFinding(t, findings, "resource-not-found-code")
	assertFindingEnforcement(t, findings, "resource-not-found-code", report.EnforcementReportOnly)
}

func TestHTTPInitializeTextFalsePositiveIsWeakWarning(t *testing.T) {
	fixture := newHTTPFixture(t, initializeMentionNoLegacyProfile)
	defer fixture.Close()

	findings, err := Analyze(Options{Transport: "http", URL: fixture.URL, SpecTarget: spec.TargetVersion})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	assertFinding(t, findings, "initialize-text-heuristic")
	assertNoFinding(t, findings, "initialize-handshake-removed")
	assertFindingSeverity(t, findings, "initialize-text-heuristic", report.SeverityWarning)
}

func TestHTTPInitializeTextFalseNegativeDoesNotInventEvidence(t *testing.T) {
	fixture := newHTTPFixture(t, legacySilentProfile)
	defer fixture.Close()

	findings, err := Analyze(Options{Transport: "http", URL: fixture.URL, SpecTarget: spec.TargetVersion})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	assertFinding(t, findings, "server-discover-required")
	assertNoFinding(t, findings, "initialize-text-heuristic")
	assertNoFinding(t, findings, "initialize-handshake-removed")
}

func TestHTTPAnalyzeRequiresURL(t *testing.T) {
	_, err := Analyze(Options{Transport: "http", SpecTarget: spec.TargetVersion})
	if err == nil {
		t.Fatal("expected missing url error")
	}
}

func TestHTTPTraceSeparatesProbeAndEvaluation(t *testing.T) {
	fixture := newHTTPFixture(t, mixedProfile)
	defer fixture.Close()

	trace, err := ProbeHTTP(Options{Transport: "http", URL: fixture.URL, SpecTarget: spec.TargetVersion})
	if err != nil {
		t.Fatalf("ProbeHTTP returned error: %v", err)
	}
	if len(trace.Observations) == 0 {
		t.Fatal("expected observations")
	}
	if trace.Observations[0].Probe == "" {
		t.Fatal("expected raw observation probe name")
	}
}

type fixtureProfile int

const (
	compliantProfile fixtureProfile = iota
	legacyProfile
	mixedProfile
	initializeMentionNoLegacyProfile
	legacySilentProfile
	resourceNotFoundProfile
)

type httpFixture struct {
	*httptest.Server
	mu      sync.Mutex
	methods []string
	profile fixtureProfile
}

func newHTTPFixture(t *testing.T, profile fixtureProfile) *httpFixture {
	t.Helper()
	fixture := &httpFixture{profile: profile}
	fixture.Server = httptest.NewServer(http.HandlerFunc(fixture.handle))
	return fixture
}

func (f *httpFixture) Methods() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.methods...)
}

func (f *httpFixture) handle(w http.ResponseWriter, r *http.Request) {
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPCError(w, http.StatusBadRequest, 0, -32700, "parse error")
		return
	}
	f.mu.Lock()
	f.methods = append(f.methods, req.Method)
	f.mu.Unlock()

	switch f.profile {
	case compliantProfile:
		f.handleCompliant(w, r, req)
	case legacyProfile:
		f.handleLegacy(w, r, req)
	case mixedProfile:
		f.handleMixed(w, r, req)
	case initializeMentionNoLegacyProfile:
		f.handleInitializeMentionNoLegacy(w, r, req)
	case legacySilentProfile:
		f.handleLegacySilent(w, r, req)
	case resourceNotFoundProfile:
		f.handleResourceNotFound(w, r, req)
	default:
		writeRPCError(w, http.StatusInternalServerError, req.ID, -32603, "unknown fixture")
	}
}

func (f *httpFixture) handleCompliant(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	if err := validate2026Headers(r, req); err != "" {
		writeRPCError(w, http.StatusBadRequest, req.ID, -32602, err)
		return
	}
	if !hasRequiredMeta(req.Params) {
		writeRPCError(w, http.StatusBadRequest, req.ID, -32602, "missing required meta")
		return
	}
	if r.Header.Get("MCP-Protocol-Version") != metaVersionFromParams(req.Params) {
		writeRPCError(w, http.StatusBadRequest, req.ID, -32602, "protocol version mismatch")
		return
	}
	writeReadOnlyResult(w, req, true)
}

func (f *httpFixture) handleLegacy(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	w.Header().Set("Mcp-Session-Id", "legacy-session-secret")
	if req.Method != "initialize" || r.Header.Get("Mcp-Session-Id") == "" {
		writeRPCError(w, http.StatusBadRequest, req.ID, -32000, "initialize first; Mcp-Session-Id required")
		return
	}
	writeRPCResult(w, req.ID, map[string]any{"protocolVersion": "2025-11-25"})
}

func (f *httpFixture) handleMixed(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	if req.Method == "server/discover" {
		writeRPCResult(w, req.ID, map[string]any{
			"supportedVersions": []string{spec.TargetVersion},
			"capabilities":      map[string]any{"tools": map[string]any{}},
			"serverInfo":        map[string]any{"name": "mixed", "version": "test"},
		})
		return
	}
	writeReadOnlyResult(w, req, false)
}

func (f *httpFixture) handleInitializeMentionNoLegacy(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	if req.Method == "tools/list" && r.Header.Get("Mcp-Method") == "" {
		writeRPCError(w, http.StatusBadRequest, req.ID, -32602, "do not initialize first; send headers")
		return
	}
	f.handleCompliant(w, r, req)
}

func (f *httpFixture) handleLegacySilent(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	writeRPCError(w, http.StatusBadRequest, req.ID, -32000, "unsupported request")
}

func (f *httpFixture) handleResourceNotFound(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	if req.Method == "resources/read" {
		writeRPCError(w, http.StatusBadRequest, req.ID, -32002, "resource not found")
		return
	}
	f.handleCompliant(w, r, req)
}

func validate2026Headers(r *http.Request, req rpcRequest) string {
	if r.Header.Get("Mcp-Method") == "" {
		return "missing Mcp-Method"
	}
	if r.Header.Get("Mcp-Method") != req.Method {
		return "Mcp-Method mismatch"
	}
	if req.Method == "resources/read" && r.Header.Get("Mcp-Name") == "" {
		return "missing Mcp-Name"
	}
	if r.Header.Get("MCP-Protocol-Version") == "" {
		return "missing protocol version header"
	}
	return ""
}

func hasRequiredMeta(params any) bool {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return false
	}
	meta, ok := paramsMap["_meta"].(map[string]any)
	if !ok {
		return false
	}
	if _, ok := meta["io.modelcontextprotocol/protocolVersion"].(string); !ok {
		return false
	}
	if _, ok := meta["io.modelcontextprotocol/clientInfo"].(map[string]any); !ok {
		return false
	}
	if _, ok := meta["io.modelcontextprotocol/clientCapabilities"].(map[string]any); !ok {
		return false
	}
	return true
}

func metaVersionFromParams(params any) string {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return ""
	}
	meta, ok := paramsMap["_meta"].(map[string]any)
	if !ok {
		return ""
	}
	version, _ := meta["io.modelcontextprotocol/protocolVersion"].(string)
	return version
}

func writeReadOnlyResult(w http.ResponseWriter, req rpcRequest, includeCache bool) {
	switch req.Method {
	case "server/discover":
		result := map[string]any{
			"supportedVersions": []string{spec.TargetVersion},
			"capabilities":      map[string]any{"tools": map[string]any{}, "resources": map[string]any{}, "prompts": map[string]any{}},
			"serverInfo":        map[string]any{"name": "compliant", "version": "test"},
		}
		writeRPCResult(w, req.ID, result)
	case "tools/list":
		result := map[string]any{"tools": []any{}}
		addCacheFields(result, includeCache)
		writeRPCResult(w, req.ID, result)
	case "resources/list":
		result := map[string]any{"resources": []any{map[string]any{"uri": "file:///fixture.txt", "name": "fixture"}}}
		addCacheFields(result, includeCache)
		writeRPCResult(w, req.ID, result)
	case "resources/read":
		result := map[string]any{"contents": []any{map[string]any{"uri": "file:///fixture.txt", "text": "fixture"}}}
		addCacheFields(result, includeCache)
		writeRPCResult(w, req.ID, result)
	case "prompts/list":
		result := map[string]any{"prompts": []any{}}
		addCacheFields(result, includeCache)
		writeRPCResult(w, req.ID, result)
	default:
		writeRPCError(w, http.StatusNotFound, req.ID, -32601, "method not found")
	}
}

func addCacheFields(result map[string]any, include bool) {
	if !include {
		return
	}
	result["ttlMs"] = float64(0)
	result["cacheScope"] = "public"
}

func writeRPCResult(w http.ResponseWriter, id int, result map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func writeRPCError(w http.ResponseWriter, status, id, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func assertFinding(t *testing.T, findings []report.Finding, rule string) {
	t.Helper()
	if !slices.ContainsFunc(findings, func(f report.Finding) bool { return f.Rule == rule }) {
		t.Fatalf("expected finding %q in %#v", rule, findings)
	}
}

func assertNoFinding(t *testing.T, findings []report.Finding, rule string) {
	t.Helper()
	if slices.ContainsFunc(findings, func(f report.Finding) bool { return f.Rule == rule }) {
		t.Fatalf("did not expect finding %q in %#v", rule, findings)
	}
}

func assertFindingSeverity(t *testing.T, findings []report.Finding, rule string, severity report.Severity) {
	t.Helper()
	for _, finding := range findings {
		if finding.Rule == rule {
			if finding.Severity != severity {
				t.Fatalf("expected severity %s for %q, got %s", severity, rule, finding.Severity)
			}
			return
		}
	}
	t.Fatalf("expected finding %q in %#v", rule, findings)
}

func assertFindingEnforcement(t *testing.T, findings []report.Finding, rule string, enforcement report.Enforcement) {
	t.Helper()
	for _, finding := range findings {
		if finding.Rule == rule {
			if finding.Enforcement != enforcement {
				t.Fatalf("expected enforcement %s for %q, got %s", enforcement, rule, finding.Enforcement)
			}
			return
		}
	}
	t.Fatalf("expected finding %q in %#v", rule, findings)
}

func assertNoToolCall(t *testing.T, methods []string) {
	t.Helper()
	if slices.Contains(methods, "tools/call") {
		t.Fatalf("analyzer sent mutating tools/call probe: %v", methods)
	}
}

func assertNoSecretLeak(t *testing.T, findings []report.Finding, secret string) {
	t.Helper()
	encoded, err := json.Marshal(findings)
	if err != nil {
		t.Fatalf("marshal findings: %v", err)
	}
	if strings.Contains(string(encoded), secret) {
		t.Fatalf("findings leaked secret %q: %s", secret, encoded)
	}
}
