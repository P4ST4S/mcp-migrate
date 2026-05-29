package report

import (
	"fmt"
	"io"
)

func WriteMarkdown(w io.Writer, findings []Finding) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(w, "# MCP Migration Report\n\nNo findings.")
		return err
	}

	if _, err := fmt.Fprintln(w, "# MCP Migration Report"); err != nil {
		return err
	}
	for _, finding := range findings {
		if _, err := fmt.Fprintf(w, "\n## %s\n\n- Severity: `%s`\n- Spec target: `%s`\n", finding.Rule, finding.Severity, finding.SpecTarget); err != nil {
			return err
		}
		if finding.SEP != "" {
			if _, err := fmt.Fprintf(w, "- SEP: `%s`\n", finding.SEP); err != nil {
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
