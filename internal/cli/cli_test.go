package cli

import (
	"bytes"
	"testing"
)

func TestAnalyzeEmitsEmptyJSONL(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"analyze", "--transport", "stdio", "--server-command", "fixture"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr %q", code, stderr.String())
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("expected empty JSONL output, got %q", got)
	}
}

func TestAnalyzeRejectsUnknownTransport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"analyze", "--transport", "websocket"}, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %q", stdout.String())
	}
}

func TestAnalyzeHTTPRequiresURL(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"analyze", "--transport", "http"}, nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d, stderr %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %q", stdout.String())
	}
}
