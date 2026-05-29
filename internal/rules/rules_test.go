package rules

import (
	"testing"

	"github.com/P4ST4S/mcp-migrate/internal/report"
)

func TestNewRegistryRejectsDuplicateIDs(t *testing.T) {
	_, err := NewRegistry([]Rule{
		{ID: "same", Severity: report.SeverityWarning, Status: StatusConfirmed},
		{ID: "same", Severity: report.SeverityInfo, Status: StatusConfirmed},
	})
	if err == nil {
		t.Fatal("expected duplicate rule error")
	}
}

func TestNewRegistryRejectsMissingSEPReference(t *testing.T) {
	_, err := NewRegistry([]Rule{{
		ID:       "missing-sep",
		Severity: report.SeverityWarning,
		Status:   StatusConfirmed,
	}})
	if err == nil {
		t.Fatal("expected missing SEP reference error")
	}
}

func TestDefaultRegistryIsSeeded(t *testing.T) {
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry returned error: %v", err)
	}
	if got := len(registry.All()); got < 25 {
		t.Fatalf("expected seeded rules, got %d", got)
	}
}

func TestSEPVerificationTagging(t *testing.T) {
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry returned error: %v", err)
	}

	tests := []struct {
		id           string
		verification report.SEPVerification
	}{
		{id: "trace-context-meta", verification: report.SEPVerified},
		{id: "protocol-version-per-request", verification: report.SEPUnverified},
		{id: "auth-application-type-dcr", verification: report.SEPUnverified},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			rule, ok := registry.Find(tt.id)
			if !ok {
				t.Fatalf("rule %q not found", tt.id)
			}
			sep := rule.SEPRef()
			if sep == nil {
				t.Fatalf("rule %q has no sep", tt.id)
			}
			if sep.Verification != tt.verification {
				t.Fatalf("expected %s, got %s", tt.verification, sep.Verification)
			}
		})
	}
}

func TestPendingVerificationIsReportOnly(t *testing.T) {
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry returned error: %v", err)
	}
	rule, ok := registry.Find("resource-not-found-code")
	if !ok {
		t.Fatal("resource-not-found-code not found")
	}
	if got := rule.Enforcement(); got != report.EnforcementReportOnly {
		t.Fatalf("expected report-only enforcement, got %s", got)
	}
	finding := rule.Finding(report.Source{Mode: "live", Ref: "fixture"})
	if finding.Enforcement != report.EnforcementReportOnly {
		t.Fatalf("expected report-only finding, got %s", finding.Enforcement)
	}
}

func TestUnverifiedSEPIsReportOnlyByDefault(t *testing.T) {
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry returned error: %v", err)
	}

	for _, id := range []string{
		"server-discover-required",
		"client-info-capabilities-per-request",
		"cacheable-results-required",
	} {
		t.Run(id, func(t *testing.T) {
			rule, ok := registry.Find(id)
			if !ok {
				t.Fatalf("rule %q not found", id)
			}
			if got := rule.SEPRef().Verification; got != report.SEPUnverified {
				t.Fatalf("expected unverified sep, got %s", got)
			}
			if got := rule.Enforcement(); got != report.EnforcementReportOnly {
				t.Fatalf("expected report-only enforcement, got %s", got)
			}
		})
	}
}

func TestVerifiedFinalSEPsRemainEnforced(t *testing.T) {
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry returned error: %v", err)
	}

	for _, id := range []string{"mcp-session-id-removed", "http-standard-headers"} {
		t.Run(id, func(t *testing.T) {
			rule, ok := registry.Find(id)
			if !ok {
				t.Fatalf("rule %q not found", id)
			}
			if got := rule.SEPRef().Verification; got != report.SEPVerified {
				t.Fatalf("expected verified sep, got %s", got)
			}
			if got := rule.Enforcement(); got != report.EnforcementEnforced {
				t.Fatalf("expected enforced, got %s", got)
			}
		})
	}
}
