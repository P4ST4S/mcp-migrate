package report

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteMarkdownEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, nil); err != nil {
		t.Fatalf("WriteMarkdown returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "No findings.") {
		t.Fatalf("expected empty report message, got %q", buf.String())
	}
}
