package rules

import "fmt"

type Severity string

const (
	SeverityBreaking   Severity = "breaking"
	SeverityDeprecated Severity = "deprecated"
	SeverityWarning    Severity = "warning"
	SeverityInfo       Severity = "info"
)

type Rule struct {
	ID          string
	SEP         string
	Severity    Severity
	AppliesTo   []string
	Autofixable bool
	Status      string
}

type Registry struct {
	rules []Rule
}

func NewRegistry(rules []Rule) (*Registry, error) {
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		if rule.ID == "" {
			return nil, fmt.Errorf("rule id is required")
		}
		if _, ok := seen[rule.ID]; ok {
			return nil, fmt.Errorf("duplicate rule id %q", rule.ID)
		}
		seen[rule.ID] = struct{}{}
	}
	return &Registry{rules: append([]Rule(nil), rules...)}, nil
}

func (r *Registry) All() []Rule {
	if r == nil {
		return nil
	}
	return append([]Rule(nil), r.rules...)
}

func DefaultRegistry() (*Registry, error) {
	return NewRegistry(nil)
}
