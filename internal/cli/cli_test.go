package cli

import (
	"bytes"
	"testing"
)

func TestAnalyzeEmitsEmptyJSONL(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"analyze", "--transport", "http", "--url", "http://localhost:3000/mcp"}, nil, &stdout, &stderr)
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
