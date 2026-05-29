package state

import (
	"encoding/json"
	"sort"
	"strings"
)

type ListObservation struct {
	Probe    string
	Method   string
	Accepted bool
	Result   map[string]any
}

type ListDrift struct {
	Method      string
	FirstProbe  string
	SecondProbe string
}

func DetectListDrift(observations []ListObservation) []ListDrift {
	type baseline struct {
		probe     string
		canonical string
	}
	seen := make(map[string]baseline)
	var drifts []ListDrift
	for _, obs := range observations {
		if !obs.Accepted || obs.Result == nil {
			continue
		}
		canonical := canonicalResult(obs.Result)
		first, ok := seen[obs.Method]
		if !ok {
			seen[obs.Method] = baseline{probe: obs.Probe, canonical: canonical}
			continue
		}
		if first.canonical != canonical {
			drifts = append(drifts, ListDrift{
				Method:      obs.Method,
				FirstProbe:  first.probe,
				SecondProbe: obs.Probe,
			})
		}
	}
	return drifts
}

func canonicalResult(result map[string]any) string {
	normalized := normalize(result)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func normalize(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			if isExplicitHandleKey(key) {
				out[key] = "<explicit-handle>"
				continue
			}
			out[key] = normalize(child)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, child := range v {
			out = append(out, normalize(child))
		}
		sort.Slice(out, func(i, j int) bool {
			left, _ := json.Marshal(out[i])
			right, _ := json.Marshal(out[j])
			return string(left) < string(right)
		})
		return out
	default:
		return value
	}
}

func isExplicitHandleKey(key string) bool {
	normalized := strings.ToLower(key)
	return normalized == "handle" ||
		normalized == "statehandle" ||
		normalized == "state_handle" ||
		normalized == "resourcehandle" ||
		normalized == "resource_handle" ||
		normalized == "continuationhandle" ||
		normalized == "continuation_handle"
}
