package static

import "github.com/P4ST4S/mcp-migrate/internal/report"

type Options struct {
	Root       string
	SpecTarget string
}

func Analyze(Options) ([]report.Finding, error) {
	return nil, nil
}
