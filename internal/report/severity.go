package report

type SeverityLegendEntry struct {
	Severity    Severity
	Description string
}

var SeverityLegend = []SeverityLegendEntry{
	{
		Severity:    SeverityBreaking,
		Description: "Incompatible with a strict MCP 2026-07-28 peer. This does not mean the feature stops working on July 28, 2026.",
	},
	{
		Severity:    SeverityDeprecated,
		Description: "Still functional in MCP 2026-07-28, but in the Deprecated lifecycle state. Deprecated features remain functional for at least 12 months before earliest removal eligibility.",
	},
	{
		Severity:    SeverityWarning,
		Description: "Operational risk or minor non-conformance that may affect portability, scaling, or future migration.",
	},
	{
		Severity:    SeverityInfo,
		Description: "Informational modernization or interoperability suggestion.",
	},
}
