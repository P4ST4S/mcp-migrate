package report

import (
	"bytes"
	"testing"
)

func TestWriteJSONLEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSONL(&buf, nil); err != nil {
		t.Fatalf("WriteJSONL returned error: %v", err)
	}
	if got := buf.String(); got != "" {
		t.Fatalf("expected empty JSONL output, got %q", got)
	}
}

func TestWriteJSONLFindings(t *testing.T) {
	findings := []Finding{{
		Schema:      "mcp-migrate/finding/v1",
		Rule:        "resource-not-found-code",
		SEP:         &SEPRef{ID: "SEP-2164", Status: "Draft", Verification: SEPUnverified},
		Severity:    SeverityBreaking,
		Enforcement: EnforcementReportOnly,
		SpecTarget:  "2026-07-28",
		Source:      Source{Mode: "live", Ref: "http://localhost:3000/mcp"},
		Message:     "Resource not found uses legacy error code.",
		Autofix:     true,
	}}

	var buf bytes.Buffer
	if err := WriteJSONL(&buf, findings); err != nil {
		t.Fatalf("WriteJSONL returned error: %v", err)
	}

	want := "{\"schema\":\"mcp-migrate/finding/v1\",\"rule\":\"resource-not-found-code\",\"sep\":{\"id\":\"SEP-2164\",\"status\":\"Draft\",\"verification\":\"unverified\"},\"severity\":\"breaking\",\"enforcement\":\"report-only\",\"spec_target\":\"2026-07-28\",\"source\":{\"mode\":\"live\",\"ref\":\"http://localhost:3000/mcp\"},\"message\":\"Resource not found uses legacy error code.\",\"autofix\":true}\n"
	if got := buf.String(); got != want {
		t.Fatalf("unexpected JSONL\nwant: %q\ngot:  %q", want, got)
	}
}
