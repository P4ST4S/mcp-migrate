package live

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/P4ST4S/mcp-migrate/internal/report"
	"github.com/P4ST4S/mcp-migrate/internal/spec"
)

func TestExamplesAreCurrent(t *testing.T) {
	examples := []struct {
		name     string
		findings []report.Finding
	}{
		{name: "http-compliant", findings: exampleHTTPFindings(t, compliantProfile)},
		{name: "http-legacy", findings: exampleHTTPFindings(t, legacyProfile)},
		{name: "http-mixed", findings: exampleHTTPFindings(t, mixedProfile)},
		{name: "http-resource-not-found", findings: exampleHTTPFindingsWithOptions(t, resourceNotFoundProfile, Options{AllowResourceRead: true})},
		{name: "http-stateful-lists", findings: exampleHTTPFindings(t, statefulListProfile)},
		{name: "http-explicit-handle-lists", findings: exampleHTTPFindings(t, explicitHandleListProfile)},
		{name: "stdio-compliant", findings: exampleSTDIOFindings(t, "compliant")},
		{name: "stdio-legacy", findings: exampleSTDIOFindings(t, "legacy")},
		{name: "stdio-mixed", findings: exampleSTDIOFindings(t, "mixed")},
		{name: "stdio-stateful-lists", findings: exampleSTDIOFindings(t, "stateful")},
		{name: "stdio-explicit-handle-lists", findings: exampleSTDIOFindings(t, "explicit-handle")},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			assertExampleFile(t, example.name+".jsonl", renderJSONL(t, example.findings))
			assertExampleFile(t, example.name+".md", renderMarkdown(t, example.findings))
		})
	}
}

func exampleHTTPFindings(t *testing.T, profile fixtureProfile) []report.Finding {
	t.Helper()
	return exampleHTTPFindingsWithOptions(t, profile, Options{})
}

func exampleHTTPFindingsWithOptions(t *testing.T, profile fixtureProfile, opts Options) []report.Finding {
	t.Helper()
	fixture := newHTTPFixture(t, profile)
	defer fixture.Close()

	opts.Transport = "http"
	opts.URL = fixture.URL
	opts.SpecTarget = spec.TargetVersion
	findings, err := Analyze(opts)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	return normalizeExampleFindings(findings, "http://example.test/mcp")
}

func exampleSTDIOFindings(t *testing.T, profile string) []report.Finding {
	t.Helper()
	trace, err := ProbeSTDIO(Options{
		Transport:     "stdio",
		ServerCommand: helperCommand(profile, "--token example-secret"),
		SpecTarget:    spec.TargetVersion,
		Timeout:       500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("ProbeSTDIO returned error: %v", err)
	}
	return normalizeExampleFindings(EvaluateSTDIOTrace(trace, mustRegistry(t)), "mcp-stdio-fixture "+profile)
}

func normalizeExampleFindings(findings []report.Finding, ref string) []report.Finding {
	out := append([]report.Finding(nil), findings...)
	for i := range out {
		out[i].Source.Ref = ref
	}
	return out
}

func renderJSONL(t *testing.T, findings []report.Finding) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := report.WriteJSONL(&buf, findings); err != nil {
		t.Fatalf("WriteJSONL returned error: %v", err)
	}
	return buf.Bytes()
}

func renderMarkdown(t *testing.T, findings []report.Finding) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := report.WriteMarkdown(&buf, findings); err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}
	return buf.Bytes()
}

func assertExampleFile(t *testing.T, name string, content []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "testdata", "examples", name)
	if os.Getenv("UPDATE_EXAMPLES") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create examples dir: %v", err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write example %s: %v", name, err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example %s: %v", name, err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("example %s is stale; run UPDATE_EXAMPLES=1 go test ./internal/analyze/live", name)
	}
}
