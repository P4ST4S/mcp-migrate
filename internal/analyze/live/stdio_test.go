package live

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/P4ST4S/mcp-migrate/internal/report"
	"github.com/P4ST4S/mcp-migrate/internal/rules"
	"github.com/P4ST4S/mcp-migrate/internal/spec"
)

func TestSTDIOAnalyzeCompliantServer(t *testing.T) {
	findings, methods := runSTDIOAnalyze(t, "compliant", "")
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
	assertNoToolCall(t, methods)
}

func TestSTDIOAnalyzeLegacyServer(t *testing.T) {
	findings, methods := runSTDIOAnalyze(t, "legacy", "--token stdio-secret")
	assertFinding(t, findings, "server-discover-required")
	assertFinding(t, findings, "initialize-handshake-removed")
	assertFindingEnforcement(t, findings, "server-discover-required", report.EnforcementReportOnly)
	assertFindingEnforcement(t, findings, "initialize-handshake-removed", report.EnforcementReportOnly)
	assertNoFinding(t, findings, "initialize-text-heuristic")
	assertNoToolCall(t, methods)
	assertNoSecretLeak(t, findings, "stdio-secret")
	assertNoSecretLeak(t, findings, "STDERR_SECRET")
}

func TestSTDIOAnalyzeMixedServer(t *testing.T) {
	findings, methods := runSTDIOAnalyze(t, "mixed", "")
	assertFinding(t, findings, "client-info-capabilities-per-request")
	assertFinding(t, findings, "cacheable-results-required")
	assertFindingEnforcement(t, findings, "client-info-capabilities-per-request", report.EnforcementReportOnly)
	assertFindingEnforcement(t, findings, "cacheable-results-required", report.EnforcementReportOnly)
	assertNoFinding(t, findings, "initialize-handshake-removed")
	assertNoToolCall(t, methods)
}

func TestSTDIOTimeoutKillsProcess(t *testing.T) {
	start := time.Now()
	findings, _ := runSTDIOAnalyzeWithTimeout(t, "hang", "", 50*time.Millisecond)
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout probe did not return promptly")
	}
	assertFinding(t, findings, "server-discover-required")
}

func runSTDIOAnalyze(t *testing.T, profile, extra string) ([]report.Finding, []string) {
	t.Helper()
	return runSTDIOAnalyzeWithTimeout(t, profile, extra, 500*time.Millisecond)
}

func runSTDIOAnalyzeWithTimeout(t *testing.T, profile, extra string, timeout time.Duration) ([]report.Finding, []string) {
	t.Helper()
	command := helperCommand(profile, extra)
	trace, err := ProbeSTDIO(Options{
		Transport:     "stdio",
		ServerCommand: command,
		SpecTarget:    spec.TargetVersion,
		Timeout:       timeout,
	})
	if err != nil {
		t.Fatalf("ProbeSTDIO returned error: %v", err)
	}
	registry := mustRegistry(t)
	return EvaluateSTDIOTrace(trace, registry), stdioMethods(trace)
}

func helperCommand(profile, extra string) string {
	parts := []string{
		strconv.Quote(os.Args[0]),
		"-test.run=TestSTDIOHelperProcess",
		"--",
		profile,
	}
	if extra != "" {
		parts = append(parts, extra)
	}
	return strings.Join(parts, " ")
}

func stdioMethods(trace STDIOTrace) []string {
	methods := make([]string, 0, len(trace.Observations))
	for _, obs := range trace.Observations {
		methods = append(methods, obs.RPCMethod)
	}
	return methods
}

func mustRegistry(t *testing.T) *rules.Registry {
	t.Helper()
	registry, err := rules.DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry returned error: %v", err)
	}
	return registry
}

func TestSTDIOHelperProcess(t *testing.T) {
	if len(os.Args) < 3 {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator == -1 || separator+1 >= len(os.Args) {
		return
	}
	profile := os.Args[separator+1]
	fmt.Fprintln(os.Stderr, "STDERR_SECRET should never be emitted")
	runSTDIOHelper(profile)
	os.Exit(0)
}

func runSTDIOHelper(profile string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	initialized := false
	for scanner.Scan() {
		var req rpcRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			writeSTDIOError(0, -32700, "parse error")
			continue
		}
		switch profile {
		case "compliant":
			handleSTDIOCompliant(req)
		case "legacy":
			initialized = handleSTDIOLegacy(req, initialized)
		case "mixed":
			handleSTDIOMixed(req)
		case "hang":
			select {}
		default:
			writeSTDIOError(req.ID, -32603, "unknown profile")
		}
	}
}

func handleSTDIOCompliant(req rpcRequest) {
	if req.Method != "server/discover" && !hasRequiredMeta(req.Params) {
		writeSTDIOError(req.ID, -32602, "missing required meta")
		return
	}
	switch req.Method {
	case "server/discover":
		writeSTDIOResult(req.ID, map[string]any{
			"supportedVersions": []string{spec.TargetVersion},
			"capabilities":      map[string]any{"tools": map[string]any{}, "resources": map[string]any{}, "prompts": map[string]any{}},
			"serverInfo":        map[string]any{"name": "stdio-compliant", "version": "test"},
		})
	case "tools/list":
		writeSTDIOResult(req.ID, cacheable(map[string]any{"tools": []any{}}))
	case "resources/list":
		writeSTDIOResult(req.ID, cacheable(map[string]any{"resources": []any{}}))
	case "prompts/list":
		writeSTDIOResult(req.ID, cacheable(map[string]any{"prompts": []any{}}))
	default:
		writeSTDIOError(req.ID, -32601, "method not found")
	}
}

func handleSTDIOLegacy(req rpcRequest, initialized bool) bool {
	if req.Method == "server/discover" {
		writeSTDIOError(req.ID, -32601, "method not found")
		return initialized
	}
	if req.Method == "initialize" {
		writeSTDIOResult(req.ID, map[string]any{
			"protocolVersion": "2025-11-25",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "stdio-legacy", "version": "test"},
		})
		return true
	}
	if !initialized {
		writeSTDIOError(req.ID, -32000, "not ready")
		return initialized
	}
	switch req.Method {
	case "tools/list":
		writeSTDIOResult(req.ID, map[string]any{"tools": []any{}})
	case "resources/list":
		writeSTDIOResult(req.ID, map[string]any{"resources": []any{}})
	case "prompts/list":
		writeSTDIOResult(req.ID, map[string]any{"prompts": []any{}})
	default:
		writeSTDIOError(req.ID, -32601, "method not found")
	}
	return initialized
}

func handleSTDIOMixed(req rpcRequest) {
	if req.Method == "server/discover" {
		writeSTDIOResult(req.ID, map[string]any{
			"supportedVersions": []string{spec.TargetVersion},
			"capabilities":      map[string]any{"tools": map[string]any{}},
			"serverInfo":        map[string]any{"name": "stdio-mixed", "version": "test"},
		})
		return
	}
	switch req.Method {
	case "tools/list":
		writeSTDIOResult(req.ID, map[string]any{"tools": []any{}})
	case "resources/list":
		writeSTDIOResult(req.ID, map[string]any{"resources": []any{}})
	case "prompts/list":
		writeSTDIOResult(req.ID, map[string]any{"prompts": []any{}})
	default:
		writeSTDIOError(req.ID, -32601, "method not found")
	}
}

func cacheable(result map[string]any) map[string]any {
	result["ttlMs"] = float64(0)
	result["cacheScope"] = "public"
	return result
}

func writeSTDIOResult(id int, result map[string]any) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func writeSTDIOError(id, code int, message string) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
