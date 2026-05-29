package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpEmitsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"help"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr %q", code, stderr.String())
	}
	if got := stdout.String(); got == "" {
		t.Fatal("expected usage output")
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

func TestHelpMentionsPatch(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	Run([]string{"help"}, nil, &stdout, &stderr)
	if !strings.Contains(stdout.String(), "patch") {
		t.Fatalf("expected help to mention patch, got %q", stdout.String())
	}
}

func TestPatchRequiresPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"patch"}, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestPatchDryRunNothingToChange(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "clean.go")
	if err := os.WriteFile(path, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"patch", "--path", path}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no patchable") {
		t.Fatalf("expected 'no patchable' message, got %q", stdout.String())
	}
}

func TestPatchDryRunReportsDiff(t *testing.T) {
	tmp := t.TempDir()
	src := `package h

func handleResourcesRead(uri string) error {
	return &rpcError{Code: -32002, Message: "resource not found"}
}
`
	path := filepath.Join(tmp, "handler.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"patch", "--path", path}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "would patch") {
		t.Errorf("expected 'would patch' in output, got %q", out)
	}
	if !strings.Contains(out, "-32002") {
		t.Errorf("expected -32002 in diff output, got %q", out)
	}
	if !strings.Contains(out, "-32602") {
		t.Errorf("expected -32602 in diff output, got %q", out)
	}
	// dry-run: file must be unchanged
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "-32002") {
		t.Error("dry-run must not modify the file on disk")
	}
}

func TestPatchWriteModifiesFile(t *testing.T) {
	tmp := t.TempDir()
	src := `package h

func handleResourcesRead(uri string) error {
	return &rpcError{Code: -32002, Message: "resource not found"}
}
`
	path := filepath.Join(tmp, "handler.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"patch", "--path", path, "--write"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", code, stderr.String())
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "-32002") {
		t.Error("expected -32002 to be replaced after --write")
	}
	if !strings.Contains(string(got), "-32602") {
		t.Error("expected -32602 after --write")
	}
}
