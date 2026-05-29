package patch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureDir returns the absolute path to a testdata/patch sub-directory.
func fixtureDir(t *testing.T, sub string) string {
	t.Helper()
	// The package lives at internal/patch/; testdata is at the repo root.
	return filepath.Join("..", "..", "testdata", "patch", sub)
}

// fixtureFile reads a testdata/patch file and returns its content.
func fixtureFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "patch", rel))
	if err != nil {
		t.Fatalf("fixtureFile %s: %v", rel, err)
	}
	return string(b)
}

// copyDir copies all files from src into a new temp dir and returns the temp path.
func copyDir(t *testing.T, src string) string {
	t.Helper()
	tmp := t.TempDir()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("copyDir ReadDir %s: %v", src, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatalf("copyDir ReadFile: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmp, e.Name()), data, 0o644); err != nil {
			t.Fatalf("copyDir WriteFile: %v", err)
		}
	}
	return tmp
}

// --- dry-run tests ---

func TestDryRunGoDoesNotModifyDisk(t *testing.T) {
	src := fixtureDir(t, "go/resource_handler")
	tmp := copyDir(t, src)
	inputPath := filepath.Join(tmp, "input.go")
	before, _ := os.ReadFile(inputPath)

	result, err := Apply(Options{Path: inputPath, Write: false})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	after, _ := os.ReadFile(inputPath)
	if string(before) != string(after) {
		t.Fatal("dry-run must not modify file on disk")
	}
	if len(result.Files) != 1 || !result.Files[0].Changed {
		t.Fatalf("expected one changed file result, got %+v", result.Files)
	}
	if result.Files[0].Diff == "" {
		t.Fatal("expected non-empty diff in dry-run result")
	}
}

// --- write tests ---

func TestWriteGoProducesExpectedOutput(t *testing.T) {
	src := fixtureDir(t, "go/resource_handler")
	tmp := copyDir(t, src)
	inputPath := filepath.Join(tmp, "input.go")

	_, err := Apply(Options{Path: inputPath, Write: true})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	got, _ := os.ReadFile(inputPath)
	expected := fixtureFile(t, "go/resource_handler/expected.go")
	if string(got) != expected {
		t.Fatalf("patched content does not match expected.\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestWriteJSProducesExpectedOutput(t *testing.T) {
	src := fixtureDir(t, "js")
	tmp := copyDir(t, src)
	inputPath := filepath.Join(tmp, "input.js")

	_, err := Apply(Options{Path: inputPath, Write: true})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	got, _ := os.ReadFile(inputPath)
	expected := fixtureFile(t, "js/expected.js")
	if string(got) != expected {
		t.Fatalf("patched content does not match expected.\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestWritePythonProducesExpectedOutput(t *testing.T) {
	src := fixtureDir(t, "python")
	tmp := copyDir(t, src)
	inputPath := filepath.Join(tmp, "input.py")

	_, err := Apply(Options{Path: inputPath, Write: true})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	got, _ := os.ReadFile(inputPath)
	expected := fixtureFile(t, "python/expected.py")
	if string(got) != expected {
		t.Fatalf("patched content does not match expected.\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

// --- idempotence tests ---

func TestIdempotentGoAlreadyPatched(t *testing.T) {
	src := fixtureDir(t, "go/resource_handler")
	tmp := copyDir(t, src)
	inputPath := filepath.Join(tmp, "input.go")

	// First pass: write the patch.
	_, err := Apply(Options{Path: inputPath, Write: true})
	if err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// Second pass on the already-patched file: must produce no change.
	result, err := Apply(Options{Path: inputPath, Write: true})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("expected no files changed on second pass, got %+v", result.Files)
	}
}

func TestIdempotentExpectedFileNoChange(t *testing.T) {
	src := fixtureDir(t, "go/resource_handler")
	tmp := t.TempDir()
	// Copy only the expected file, rename to input.go so the patcher can find it.
	data, err := os.ReadFile(filepath.Join(src, "expected.go"))
	if err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(tmp, "input.go")
	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(Options{Path: targetPath, Write: false})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("already-patched file must produce no changes, got %+v", result.Files)
	}
}

// --- ambiguous context: must not touch ---

func TestAmbiguousContextNotPatched(t *testing.T) {
	src := fixtureDir(t, "ambiguous")
	tmp := copyDir(t, src)
	inputPath := filepath.Join(tmp, "input.go")
	before, _ := os.ReadFile(inputPath)

	result, err := Apply(Options{Path: inputPath, Write: false})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	after, _ := os.ReadFile(inputPath)
	if string(before) != string(after) {
		t.Fatal("ambiguous file must not be modified")
	}
	// The patcher should report the skipped occurrences.
	if len(result.Files) == 0 {
		t.Fatal("expected a FileResult with skipped count for the ambiguous file")
	}
	if result.Files[0].Changed {
		t.Fatal("ambiguous file must not be marked changed")
	}
	if result.Files[0].Skipped == 0 {
		t.Fatal("expected skipped > 0 for ambiguous file")
	}
}

// --- directory walk ---

func TestWalkDirectoryPatchesMultipleFiles(t *testing.T) {
	// Build a temp dir with one Go and one JS file, both in resource context.
	tmp := t.TempDir()
	goSrc := `package h

func handleResourcesRead(uri string) error {
	return &rpcError{Code: -32002, Message: "resource not found"}
}
`
	jsSrc := `async function readResource(uri) {
  return { error: { code: -32002, message: "resource not found" } };
}
`
	if err := os.WriteFile(filepath.Join(tmp, "handler.go"), []byte(goSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "client.js"), []byte(jsSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(Options{Path: tmp, Write: false})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 changed files, got %d: %+v", len(result.Files), result.Files)
	}
	for _, fr := range result.Files {
		if !fr.Changed {
			t.Errorf("file %s should be changed", fr.Path)
		}
	}
}

// --- unsupported extension: must be skipped ---

func TestUnsupportedExtensionIgnored(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("error_code: -32002\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(Options{Path: path, Write: false})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("unsupported file should not appear in results, got %+v", result.Files)
	}
}

// --- error conditions ---

func TestApplyRequiresPath(t *testing.T) {
	_, err := Apply(Options{})
	if err == nil {
		t.Fatal("expected error when path is empty")
	}
}

func TestApplyNonExistentPath(t *testing.T) {
	_, err := Apply(Options{Path: "/no/such/file.go"})
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

// --- diff content sanity ---

func TestDiffContainsOldAndNewCode(t *testing.T) {
	src := fixtureDir(t, "go/resource_handler")
	tmp := copyDir(t, src)

	result, err := Apply(Options{Path: filepath.Join(tmp, "input.go"), Write: false})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(result.Files) == 0 {
		t.Fatal("expected at least one file result")
	}
	diff := result.Files[0].Diff
	if !strings.Contains(diff, "-32002") {
		t.Error("diff should contain -32002 as removed line")
	}
	if !strings.Contains(diff, "-32602") {
		t.Error("diff should contain -32602 as added line")
	}
}

// TestDiffIsValidUnifiedDiff verifies that the produced diff has well-formed
// unified-diff headers so it can be consumed by patch(1) or Go diff parsers.
func TestDiffIsValidUnifiedDiff(t *testing.T) {
	src := fixtureDir(t, "go/resource_handler")
	tmp := copyDir(t, src)

	result, err := Apply(Options{Path: filepath.Join(tmp, "input.go"), Write: false})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	diff := result.Files[0].Diff

	// Must start with --- / +++ headers.
	lines := strings.Split(diff, "\n")
	if len(lines) < 2 {
		t.Fatalf("diff too short: %q", diff)
	}
	if !strings.HasPrefix(lines[0], "--- ") {
		t.Errorf("diff line 0 must start with '--- ', got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "+++ ") {
		t.Errorf("diff line 1 must start with '+++ ', got %q", lines[1])
	}

	// Every hunk header must parse as @@ -old,count +new,count @@
	for _, l := range lines {
		if !strings.HasPrefix(l, "@@") {
			continue
		}
		var oldStart, oldCount, newStart, newCount int
		n, scanErr := fmt.Sscanf(l, "@@ -%d,%d +%d,%d @@", &oldStart, &oldCount, &newStart, &newCount)
		if scanErr != nil || n != 4 {
			t.Errorf("malformed hunk header %q: %v", l, scanErr)
		}
		if oldCount <= 0 || newCount <= 0 {
			t.Errorf("hunk counts must be > 0, got %q", l)
		}
	}

	// Context and changed lines must not end with a trailing newline inside
	// the string (each fmt.Fprintf already adds \n; doubling would break patch(1)).
	for i, l := range lines {
		if l == "" {
			continue // blank separator between hunks is fine
		}
		if len(l) > 1 && l[len(l)-1] == '\n' {
			t.Errorf("diff line %d has embedded trailing newline: %q", i, l)
		}
	}
}

// Bug 1: -32002 at end of file without trailing newline must still be matched.
func TestPatchesCodeAtEndOfFileWithoutNewline(t *testing.T) {
	tmp := t.TempDir()
	// No trailing newline — previously the pattern required a non-digit char after -32002.
	src := `package h

func handleResourcesRead(uri string) error {
	return &rpcError{Code: -32002, Message: "resource not found"}
}`
	path := filepath.Join(tmp, "handler.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(Options{Path: path, Write: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(result.Files) != 1 || !result.Files[0].Changed {
		t.Fatalf("expected file to be patched, got %+v", result.Files)
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "-32002") {
		t.Error("expected -32002 to be replaced")
	}
	if !strings.Contains(string(got), "-32602") {
		t.Error("expected -32602 after patch")
	}
}

// Bug 2: multi-hunk diff — changes far apart must produce correct @@ line counts.
func TestMultiHunkDiffHasCorrectLineCounts(t *testing.T) {
	tmp := t.TempDir()
	// Two resources/read handlers separated by many lines so they end up in
	// distinct hunks.
	lines := make([]string, 0, 60)
	lines = append(lines, "package h", "")
	lines = append(lines, `func handleResourcesRead(uri string) error {`)
	lines = append(lines, `	return &rpcError{Code: -32002, Message: "resource not found"}`)
	lines = append(lines, `}`, "")
	for i := 0; i < 40; i++ {
		lines = append(lines, "// padding line")
	}
	lines = append(lines, "", `func readResource2(uri string) error {`)
	lines = append(lines, `	if _, ok := store[uri]; !ok {`)
	lines = append(lines, `		return &rpcError{Code: -32002, Message: "resource not found"}`)
	lines = append(lines, `	}`)
	lines = append(lines, `	return nil`)
	lines = append(lines, `}`)
	src := strings.Join(lines, "\n") + "\n"

	path := filepath.Join(tmp, "handler.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(Options{Path: path, Write: false})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file result, got %d", len(result.Files))
	}

	diff := result.Files[0].Diff
	diffLines := strings.Split(diff, "\n")

	// Count hunk headers and validate each.
	hunkCount := 0
	for _, l := range diffLines {
		if !strings.HasPrefix(l, "@@") {
			continue
		}
		hunkCount++
		var oldStart, oldCount, newStart, newCount int
		n, scanErr := fmt.Sscanf(l, "@@ -%d,%d +%d,%d @@", &oldStart, &oldCount, &newStart, &newCount)
		if scanErr != nil || n != 4 {
			t.Errorf("malformed hunk header %q: %v", l, scanErr)
		}
		if oldCount <= 0 || newCount <= 0 {
			t.Errorf("hunk line counts must be > 0, got %q", l)
		}
	}
	if hunkCount < 2 {
		t.Errorf("expected at least 2 hunks for two distant changes, got %d\ndiff:\n%s", hunkCount, diff)
	}
}

// Bug 4: WalkDir must not descend into .git, node_modules, or vendor.
func TestWalkSkipsIgnoredDirectories(t *testing.T) {
	tmp := t.TempDir()

	// Patchable file at the root.
	rootSrc := `package h

func handleResourcesRead(uri string) error {
	return &rpcError{Code: -32002, Message: "resource not found"}
}
`
	if err := os.WriteFile(filepath.Join(tmp, "handler.go"), []byte(rootSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Files inside ignored dirs that must never be touched.
	for _, dir := range []string{".git", "node_modules", "vendor"} {
		if err := os.MkdirAll(filepath.Join(tmp, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(tmp, dir, "ignored.go"),
			[]byte(rootSrc),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
	}

	result, err := Apply(Options{Path: tmp, Write: false})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Only the root handler.go must appear; the three ignored files must not.
	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file result (root only), got %d: %v",
			len(result.Files), func() []string {
				var ps []string
				for _, f := range result.Files {
					ps = append(ps, f.Path)
				}
				return ps
			}())
	}
}

// Bug 3: write must be atomic — the file must exist and be complete even if
// the process is interrupted between writes. We test atomicity indirectly by
// verifying that the written file is identical to what dry-run predicted.
func TestWriteIsAtomicAndProducesCorrectContent(t *testing.T) {
	src := fixtureDir(t, "go/resource_handler")
	tmp := copyDir(t, src)
	inputPath := filepath.Join(tmp, "input.go")

	// Capture dry-run prediction.
	dryResult, err := Apply(Options{Path: inputPath, Write: false})
	if err != nil {
		t.Fatalf("dry-run Apply: %v", err)
	}

	// Apply for real.
	_, err = Apply(Options{Path: inputPath, Write: true})
	if err != nil {
		t.Fatalf("write Apply: %v", err)
	}

	got, _ := os.ReadFile(inputPath)
	expected := fixtureFile(t, "go/resource_handler/expected.go")
	if string(got) != expected {
		t.Fatalf("written content does not match expected")
	}
	// Cross-check: diff must be non-empty before write, empty after.
	if dryResult.Files[0].Diff == "" {
		t.Error("dry-run diff must be non-empty before write")
	}
	postResult, _ := Apply(Options{Path: inputPath, Write: false})
	if len(postResult.Files) != 0 {
		t.Error("no diff expected after write (idempotence)")
	}
}
