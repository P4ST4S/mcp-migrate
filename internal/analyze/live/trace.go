package live

import (
	"net/url"
	"strings"
)

type HTTPTrace struct {
	Endpoint            string
	AllowMutatingProbes bool
	Observations        []HTTPObservation
}

type HTTPObservation struct {
	Probe                  string
	RPCMethod              string
	ReadOnly               bool
	Mutating               bool
	SentMethodHeader       bool
	SentNameHeader         bool
	HeaderBodyMismatch     bool
	MetaIncluded           bool
	MetaProtocolVersion    string
	HeaderProtocolVersion  string
	StatusCode             int
	RPCErrorCode           int
	HasRPCError            bool
	HasResult              bool
	Result                 map[string]any
	NetworkError           bool
	ParseError             bool
	HasMcpSessionID        bool
	BodyMentionsSessionID  bool
	BodyMentionsInitialize bool
}

func (o HTTPObservation) Accepted() bool {
	return !o.NetworkError && !o.ParseError && o.StatusCode >= 200 && o.StatusCode < 300 && !o.HasRPCError
}

func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<invalid-url>"
	}
	if u.User != nil {
		u.User = url.UserPassword("redacted", "redacted")
	}
	q := u.Query()
	for key := range q {
		if isSensitiveName(key) {
			q.Set(key, "redacted")
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func isSensitiveName(name string) bool {
	normalized := strings.ToLower(name)
	sensitiveParts := []string{
		"authorization",
		"token",
		"secret",
		"password",
		"passwd",
		"api-key",
		"apikey",
		"key",
		"credential",
		"cookie",
		"session",
		"code",
	}
	for _, part := range sensitiveParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}
