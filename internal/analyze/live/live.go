package live

import (
	"fmt"
	"net/http"
	"time"

	"github.com/P4ST4S/mcp-migrate/internal/report"
	"github.com/P4ST4S/mcp-migrate/internal/rules"
)

type Options struct {
	Transport           string
	URL                 string
	ServerCommand       string
	SpecTarget          string
	AllowMutatingProbes bool
	AllowResourceRead   bool
	HTTPClient          *http.Client
	Timeout             time.Duration
}

func Analyze(opts Options) ([]report.Finding, error) {
	switch opts.Transport {
	case "http":
		registry, err := rules.DefaultRegistry()
		if err != nil {
			return nil, err
		}
		trace, err := ProbeHTTP(opts)
		if err != nil {
			return nil, err
		}
		return EvaluateHTTPTrace(trace, registry), nil
	case "stdio":
		registry, err := rules.DefaultRegistry()
		if err != nil {
			return nil, err
		}
		trace, err := ProbeSTDIO(opts)
		if err != nil {
			return nil, err
		}
		return EvaluateSTDIOTrace(trace, registry), nil
	default:
		return nil, fmt.Errorf("unsupported transport %q", opts.Transport)
	}
}
