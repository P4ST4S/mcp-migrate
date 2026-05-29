package report

type Severity string

const (
	SeverityBreaking   Severity = "breaking"
	SeverityDeprecated Severity = "deprecated"
	SeverityWarning    Severity = "warning"
	SeverityInfo       Severity = "info"
)

type Source struct {
	Mode string `json:"mode"`
	Ref  string `json:"ref,omitempty"`
}

type Enforcement string

const (
	EnforcementEnforced   Enforcement = "enforced"
	EnforcementReportOnly Enforcement = "report-only"
)

type SEPVerification string

const (
	SEPVerified   SEPVerification = "verified"
	SEPUnverified SEPVerification = "unverified"
)

type SEPRef struct {
	ID           string          `json:"id"`
	Status       string          `json:"status,omitempty"`
	Verification SEPVerification `json:"verification"`
	Source       string          `json:"source,omitempty"`
}

type Finding struct {
	Schema      string      `json:"schema"`
	Rule        string      `json:"rule"`
	SEP         *SEPRef     `json:"sep,omitempty"`
	Severity    Severity    `json:"severity"`
	Enforcement Enforcement `json:"enforcement"`
	SpecTarget  string      `json:"spec_target"`
	Source      Source      `json:"source"`
	Message     string      `json:"message"`
	Detail      string      `json:"detail,omitempty"`
	Remediation string      `json:"remediation,omitempty"`
	Autofix     bool        `json:"autofix"`
	Status      string      `json:"status,omitempty"`
}
