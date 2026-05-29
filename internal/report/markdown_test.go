package report

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteMarkdownEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, nil); err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "No findings.") {
		t.Fatalf("expected empty report message, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "Severity Legend") {
		t.Fatalf("expected severity legend, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "at least 12 months") {
		t.Fatalf("expected deprecation window text, got %q", buf.String())
	}
}

func TestWriteMarkdownFindingShowsLocalEnforcementAndSEPVerification(t *testing.T) {
	findings := []Finding{{
		Schema:      "mcp-migrate/finding/v1",
		Rule:        "resource-not-found-code",
		SEP:         &SEPRef{ID: "SEP-2164", Status: "Draft", Verification: SEPUnverified},
		Severity:    SeverityBreaking,
		Enforcement: EnforcementReportOnly,
		SpecTarget:  "2026-07-28",
		Source:      Source{Mode: "live", Ref: "fixture"},
		Message:     "Resource not found uses a legacy error code.",
	}}

	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, findings); err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"## resource-not-found-code",
		"- Severity: `breaking`",
		"- Enforcement: `report-only`",
		"- SEP: `SEP-2164` (`Draft`, `unverified`)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected markdown to contain %q, got %q", want, out)
		}
	}
}
