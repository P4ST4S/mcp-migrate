package live

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/P4ST4S/mcp-migrate/internal/spec"
)

const defaultHTTPTimeout = 5 * time.Second

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type httpProbe struct {
	name                 string
	method               string
	params               map[string]any
	readOnly             bool
	mutating             bool
	sendMethodHeader     bool
	sendNameHeader       bool
	methodHeaderOverride string
	nameHeaderValue      string
	protocolHeader       string
}

func ProbeHTTP(opts Options) (HTTPTrace, error) {
	if opts.URL == "" {
		return HTTPTrace{}, fmt.Errorf("--url is required for http analysis")
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultHTTPTimeout
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	analyzer := httpAnalyzer{
		client:     client,
		endpoint:   opts.URL,
		specTarget: opts.SpecTarget,
	}
	if analyzer.specTarget == "" {
		analyzer.specTarget = spec.TargetVersion
	}

	trace := HTTPTrace{
		Endpoint:            sanitizeURL(opts.URL),
		AllowMutatingProbes: opts.AllowMutatingProbes,
		AllowResourceRead:   opts.AllowResourceRead,
	}
	for _, probe := range defaultHTTPProbes(analyzer.specTarget) {
		if probe.mutating && !opts.AllowMutatingProbes {
			continue
		}
		observation := analyzer.runProbe(context.Background(), probe)
		trace.Observations = append(trace.Observations, observation)
	}

	if opts.AllowResourceRead {
		// resources/read is behind an explicit opt-in. Some servers may mark
		// resources as consumed, fetch remote data, or otherwise attach side
		// effects to reads even when the protocol method name says "read".
		uri := firstResourceURI(trace)
		if uri == "" {
			return trace, nil
		}
		probe := resourceReadProbe(analyzer.specTarget, uri)
		trace.Observations = append(trace.Observations, analyzer.runProbe(context.Background(), probe))
	}

	for _, probe := range stateHTTPProbes(analyzer.specTarget, "repeat") {
		trace.Observations = append(trace.Observations, analyzer.runProbe(context.Background(), probe))
	}
	freshAnalyzer := httpAnalyzer{
		client: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{DisableKeepAlives: true},
		},
		endpoint:   opts.URL,
		specTarget: analyzer.specTarget,
	}
	for _, probe := range stateHTTPProbes(analyzer.specTarget, "fresh-client") {
		trace.Observations = append(trace.Observations, freshAnalyzer.runProbe(context.Background(), probe))
	}

	return trace, nil
}

type httpAnalyzer struct {
	client     *http.Client
	endpoint   string
	specTarget string
	nextID     int
}

func (a *httpAnalyzer) runProbe(ctx context.Context, probe httpProbe) HTTPObservation {
	a.nextID++
	observation := HTTPObservation{
		Probe:                 probe.name,
		RPCMethod:             probe.method,
		ReadOnly:              probe.readOnly,
		Mutating:              probe.mutating,
		SentMethodHeader:      probe.sendMethodHeader,
		SentNameHeader:        probe.sendNameHeader,
		HeaderBodyMismatch:    probe.methodHeaderOverride != "" && probe.methodHeaderOverride != probe.method,
		MetaIncluded:          hasMeta(probe.params),
		MetaProtocolVersion:   metaProtocolVersion(probe.params),
		HeaderProtocolVersion: probe.protocolHeader,
	}

	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      a.nextID,
		Method:  probe.method,
		Params:  probe.params,
	})
	if err != nil {
		observation.NetworkError = true
		return observation
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(body))
	if err != nil {
		observation.NetworkError = true
		return observation
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if probe.protocolHeader != "" {
		req.Header.Set("MCP-Protocol-Version", probe.protocolHeader)
	}
	if probe.sendMethodHeader {
		methodHeader := probe.method
		if probe.methodHeaderOverride != "" {
			methodHeader = probe.methodHeaderOverride
		}
		req.Header.Set("Mcp-Method", methodHeader)
	}
	if probe.sendNameHeader {
		req.Header.Set("Mcp-Name", probe.nameHeaderValue)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		observation.NetworkError = true
		return observation
	}
	defer resp.Body.Close()

	observation.StatusCode = resp.StatusCode
	observation.HasMcpSessionID = resp.Header.Get("Mcp-Session-Id") != ""

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		observation.NetworkError = true
		return observation
	}
	lowerBody := strings.ToLower(string(responseBody))
	observation.BodyMentionsSessionID = strings.Contains(lowerBody, "mcp-session-id")
	observation.BodyMentionsInitialize = strings.Contains(lowerBody, "initialize")

	if len(bytes.TrimSpace(responseBody)) == 0 {
		return observation
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(responseBody, &rpcResp); err != nil {
		observation.ParseError = true
		return observation
	}
	if rpcResp.Error != nil {
		observation.HasRPCError = true
		observation.RPCErrorCode = rpcResp.Error.Code
		return observation
	}
	if len(rpcResp.Result) > 0 {
		observation.HasResult = true
		var result map[string]any
		if err := json.Unmarshal(rpcResp.Result, &result); err == nil {
			observation.Result = result
		}
	}
	return observation
}

func defaultHTTPProbes(specTarget string) []httpProbe {
	return []httpProbe{
		{
			name:             "discover",
			method:           "server/discover",
			params:           paramsWithMeta(specTarget),
			readOnly:         true,
			sendMethodHeader: true,
			protocolHeader:   specTarget,
		},
		{
			name:             "discover-version-mismatch",
			method:           "server/discover",
			params:           paramsWithMeta(specTarget),
			readOnly:         true,
			sendMethodHeader: true,
			protocolHeader:   "2025-11-25",
		},
		{
			name:             "tools-list",
			method:           "tools/list",
			params:           paramsWithMeta(specTarget),
			readOnly:         true,
			sendMethodHeader: true,
			protocolHeader:   specTarget,
		},
		{
			name:           "tools-list-missing-method-header",
			method:         "tools/list",
			params:         paramsWithMeta(specTarget),
			readOnly:       true,
			protocolHeader: specTarget,
		},
		{
			name:                 "tools-list-mismatched-method-header",
			method:               "tools/list",
			params:               paramsWithMeta(specTarget),
			readOnly:             true,
			sendMethodHeader:     true,
			methodHeaderOverride: "resources/list",
			protocolHeader:       specTarget,
		},
		{
			name:             "tools-list-missing-meta",
			method:           "tools/list",
			params:           map[string]any{},
			readOnly:         true,
			sendMethodHeader: true,
			protocolHeader:   specTarget,
		},
		{
			name:             "resources-list",
			method:           "resources/list",
			params:           paramsWithMeta(specTarget),
			readOnly:         true,
			sendMethodHeader: true,
			protocolHeader:   specTarget,
		},
		{
			name:             "prompts-list",
			method:           "prompts/list",
			params:           paramsWithMeta(specTarget),
			readOnly:         true,
			sendMethodHeader: true,
			protocolHeader:   specTarget,
		},
	}
}

func resourceReadProbe(specTarget, uri string) httpProbe {
	params := paramsWithMeta(specTarget)
	params["uri"] = uri
	return httpProbe{
		name:             "resources-read",
		method:           "resources/read",
		params:           params,
		readOnly:         true,
		sendMethodHeader: true,
		sendNameHeader:   true,
		nameHeaderValue:  uri,
		protocolHeader:   specTarget,
	}
}

func stateHTTPProbes(specTarget, suffix string) []httpProbe {
	probes := make([]httpProbe, 0, 3)
	for _, method := range []string{"tools/list", "resources/list", "prompts/list"} {
		name := strings.ReplaceAll(method, "/", "-")
		probes = append(probes, httpProbe{
			name:             "state-" + name + "-" + suffix,
			method:           method,
			params:           paramsWithMeta(specTarget),
			readOnly:         true,
			sendMethodHeader: true,
			protocolHeader:   specTarget,
		})
	}
	return probes
}

func paramsWithMeta(specTarget string) map[string]any {
	return map[string]any{
		"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion":    specTarget,
			"io.modelcontextprotocol/clientInfo":         map[string]any{"name": "mcp-migrate", "version": "0.1.0"},
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		},
	}
}

func hasMeta(params map[string]any) bool {
	_, ok := params["_meta"]
	return ok
}

func metaProtocolVersion(params map[string]any) string {
	meta, ok := params["_meta"].(map[string]any)
	if !ok {
		return ""
	}
	version, _ := meta["io.modelcontextprotocol/protocolVersion"].(string)
	return version
}

func firstResourceURI(trace HTTPTrace) string {
	for _, obs := range trace.Observations {
		if obs.Probe != "resources-list" || !obs.Accepted() {
			continue
		}
		resources, ok := obs.Result["resources"].([]any)
		if !ok || len(resources) == 0 {
			return ""
		}
		first, ok := resources[0].(map[string]any)
		if !ok {
			return ""
		}
		uri, _ := first["uri"].(string)
		return uri
	}
	return ""
}
