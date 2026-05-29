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

type Finding struct {
	Schema      string   `json:"schema"`
	Rule        string   `json:"rule"`
	SEP         string   `json:"sep,omitempty"`
	Severity    Severity `json:"severity"`
	SpecTarget  string   `json:"spec_target"`
	Source      Source   `json:"source"`
	Message     string   `json:"message"`
	Detail      string   `json:"detail,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	Autofix     bool     `json:"autofix"`
	Status      string   `json:"status,omitempty"`
}
