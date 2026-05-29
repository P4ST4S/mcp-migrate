package report

import (
	"encoding/json"
	"fmt"
	"io"
)

func WriteJSONL(w io.Writer, findings []Finding) error {
	enc := json.NewEncoder(w)
	for _, finding := range findings {
		if err := enc.Encode(finding); err != nil {
			return fmt.Errorf("write finding: %w", err)
		}
	}
	return nil
}
