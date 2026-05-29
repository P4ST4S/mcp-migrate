package report

import (
	"fmt"
	"io"
)

func WriteMarkdown(w io.Writer, findings []Finding) error {
	if _, err := fmt.Fprintln(w, "# MCP Migration Report"); err != nil {
		return err
	}
	if err := writeSeverityLegend(w); err != nil {
		return err
	}

	if len(findings) == 0 {
		_, err := fmt.Fprintln(w, "\nNo findings.")
		return err
	}

	for _, finding := range findings {
		if _, err := fmt.Fprintf(w, "\n## %s\n\n- Severity: `%s`\n- Spec target: `%s`\n", finding.Rule, finding.Severity, finding.SpecTarget); err != nil {
			return err
		}
		if finding.Enforcement != "" {
			if _, err := fmt.Fprintf(w, "- Enforcement: `%s`\n", finding.Enforcement); err != nil {
				return err
			}
		}
		if finding.SEP != nil {
			if _, err := fmt.Fprintf(w, "- SEP: `%s` (`%s`, `%s`)\n", finding.SEP.ID, finding.SEP.Status, finding.SEP.Verification); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "\n%s\n", finding.Message); err != nil {
			return err
		}
		if finding.Detail != "" {
			if _, err := fmt.Fprintf(w, "\n%s\n", finding.Detail); err != nil {
				return err
			}
		}
		if finding.Remediation != "" {
			if _, err := fmt.Fprintf(w, "\nRemediation: %s\n", finding.Remediation); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeSeverityLegend(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "\n## Severity Legend"); err != nil {
		return err
	}
	for _, entry := range SeverityLegend {
		if _, err := fmt.Fprintf(w, "\n- `%s`: %s\n", entry.Severity, entry.Description); err != nil {
			return err
		}
	}
	return nil
}
