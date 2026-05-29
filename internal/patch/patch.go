package patch

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/P4ST4S/mcp-migrate/internal/rules"
)

// Options controls a patch run.
type Options struct {
	// Path is the file or directory to patch. Required.
	Path string
	// Write commits changes to disk. When false (default) the run is dry-run:
	// diffs are produced but no file is modified.
	Write bool
	// AllowPending permits rewriting code governed by rules whose underlying
	// SEP has not yet reached Final status. Without this flag, patch refuses
	// to modify files for pending-verification rules and returns an error
	// explaining which rules are affected and why.
	AllowPending bool
	// Registry is the rule registry used to look up patch-relevant rule
	// metadata (autofixability, SEP status). When nil, DefaultRegistry() is
	// used.
	Registry *rules.Registry
}

// FileResult records the outcome for one file.
type FileResult struct {
	// Path is the file that was examined.
	Path string
	// Changed is true when at least one safe substitution was found.
	Changed bool
	// Skipped reports occurrences of -32002 that were not substituted because
	// the surrounding context did not confirm a resource-not-found handler.
	Skipped int
	// Diff is a unified diff string (non-empty only when Changed is true).
	Diff string
}

// Result is the aggregate output of a patch run.
type Result struct {
	Files []FileResult
	// PendingWarning is non-empty when AllowPending was required and used.
	// It describes which rules are still pending-verification.
	PendingWarning string
}

// ignoredDirs are directory names that are never descended into during a walk.
var ignoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".hg":          true,
	".svn":         true,
}

const resourceNotFoundRuleID = "resource-not-found-code"

// Apply scans Path for patchable occurrences of -32002 and, when Write is
// true, rewrites the affected files atomically. It always returns a diff
// regardless of Write.
//
// If the rule governing the substitution is pending-verification and
// opts.AllowPending is false, Apply returns an error rather than silently
// modifying code based on a non-final spec.
func Apply(opts Options) (Result, error) {
	if opts.Path == "" {
		return Result{}, fmt.Errorf("patch: path is required")
	}

	reg := opts.Registry
	if reg == nil {
		var err error
		reg, err = rules.DefaultRegistry()
		if err != nil {
			return Result{}, fmt.Errorf("patch: load registry: %w", err)
		}
	}

	rule, ok := reg.Find(resourceNotFoundRuleID)
	if !ok {
		return Result{}, fmt.Errorf("patch: rule %q not found in registry", resourceNotFoundRuleID)
	}

	// Gate: refuse to patch when the rule is pending-verification and the
	// caller has not explicitly opted in with AllowPending.
	if rule.Status == rules.StatusPendingVerification && !opts.AllowPending {
		sep := rule.SEPRef()
		sepDesc := ""
		if sep != nil {
			sepDesc = fmt.Sprintf(" (SEP %s, status: %s)", sep.ID, sep.Status)
		}
		return Result{}, fmt.Errorf(
			"patch: rule %q is pending-verification%s — "+
				"the underlying spec has not yet reached Final status.\n"+
				"Re-run with --allow-pending to apply this patch anyway.\n"+
				"Warning: modifying code based on a Draft spec carries the risk "+
				"of a second migration if the spec changes before finalisation.",
			resourceNotFoundRuleID, sepDesc,
		)
	}

	info, err := os.Stat(opts.Path)
	if err != nil {
		return Result{}, fmt.Errorf("patch: stat %s: %w", opts.Path, err)
	}

	var paths []string
	if info.IsDir() {
		if err := filepath.WalkDir(opts.Path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if ignoredDirs[d.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if isSupportedSource(p) {
				paths = append(paths, p)
			}
			return nil
		}); err != nil {
			return Result{}, fmt.Errorf("patch: walk %s: %w", opts.Path, err)
		}
	} else {
		if !isSupportedSource(opts.Path) {
			return Result{}, nil
		}
		paths = []string{opts.Path}
	}

	var result Result
	if rule.Status == rules.StatusPendingVerification && opts.AllowPending {
		sep := rule.SEPRef()
		sepDesc := ""
		if sep != nil {
			sepDesc = fmt.Sprintf("SEP %s is %s", sep.ID, sep.Status)
		}
		result.PendingWarning = fmt.Sprintf(
			"Warning: applying patch for rule %q which is pending-verification (%s). "+
				"The underlying spec may change before finalisation.",
			resourceNotFoundRuleID, sepDesc,
		)
	}

	for _, p := range paths {
		fr, err := patchFile(p, opts.Write)
		if err != nil {
			return Result{}, err
		}
		if fr.Changed || fr.Skipped > 0 {
			result.Files = append(result.Files, fr)
		}
	}
	return result, nil
}

// isSupportedSource returns true for file extensions the patcher handles.
func isSupportedSource(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".py":
		return true
	}
	return false
}

// errorCodePattern matches the literal integer -32002 as a standalone token.
// Groups 1 and 2 capture the surrounding non-digit characters so they can be
// preserved in the substitution, including at end-of-line and end-of-file.
var errorCodePattern = regexp.MustCompile(`(^|[^0-9])-32002($|[^0-9])`)

// resourceSignalPattern matches a strong, local resource-not-found signal.
// It must appear on the same line as -32002 or within 2 adjacent lines (the
// expression block), in non-comment code. The pattern intentionally requires
// a compound phrase or exact method name, not the bare word "resource":
//   - "resource not found" / "resource_not_found" as an error message literal
//   - "ResourceNotFound" as a camel-case error symbol
//   - The exact MCP method name "resources/read" on the same expression line
var resourceSignalPattern = regexp.MustCompile(
	`(?i)resource[_ -]not[_ -]found|ResourceNotFound|resources/read`,
)

// commentPrefixPattern matches lines that are pure comments in any supported
// language. Such lines are excluded from the local signal search.
var commentPrefixPattern = regexp.MustCompile(`^\s*(//|#|\*|/\*)`)

// localContextLines is the half-width of the expression window: only lines
// within this distance from the -32002 occurrence, and only non-comment lines,
// are considered for the resource signal.
const localContextLines = 2

func patchFile(path string, write bool) (FileResult, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return FileResult{}, fmt.Errorf("patch: read %s: %w", path, err)
	}

	lines := bytes.Split(src, []byte("\n"))

	// Indices (0-based) of lines that contain -32002.
	var matchLines []int
	for i, line := range lines {
		if errorCodePattern.Match(line) {
			matchLines = append(matchLines, i)
		}
	}
	if len(matchLines) == 0 {
		return FileResult{Path: path}, nil
	}

	// Determine which occurrences are safe to replace.
	// Safe = a resource-not-found signal appears on the same line or within
	// localContextLines lines, in a non-comment line.
	type decision struct {
		lineIdx int
		safe    bool
	}
	decisions := make([]decision, len(matchLines))
	for k, idx := range matchLines {
		lo := idx - localContextLines
		if lo < 0 {
			lo = 0
		}
		hi := idx + localContextLines
		if hi >= len(lines) {
			hi = len(lines) - 1
		}
		safe := false
		for i := lo; i <= hi; i++ {
			line := lines[i]
			if commentPrefixPattern.Match(line) {
				continue // ignore comment lines
			}
			if resourceSignalPattern.Match(line) {
				safe = true
				break
			}
		}
		decisions[k] = decision{lineIdx: idx, safe: safe}
	}

	safeCount := 0
	skippedCount := 0
	for _, d := range decisions {
		if d.safe {
			safeCount++
		} else {
			skippedCount++
		}
	}
	if safeCount == 0 {
		return FileResult{Path: path, Skipped: skippedCount}, nil
	}

	// Build the patched content line by line, replacing only safe occurrences.
	newLines := make([][]byte, len(lines))
	copy(newLines, lines)
	for _, d := range decisions {
		if !d.safe {
			continue
		}
		newLines[d.lineIdx] = errorCodePattern.ReplaceAll(newLines[d.lineIdx], []byte("${1}-32602${2}"))
	}
	patched := bytes.Join(newLines, []byte("\n"))

	diff := unifiedDiff(path, src, patched)

	if write {
		if err := writeAtomic(path, patched); err != nil {
			return FileResult{}, err
		}
	}

	return FileResult{
		Path:    path,
		Changed: true,
		Skipped: skippedCount,
		Diff:    diff,
	}, nil
}

// writeAtomic writes data to a temporary file in the same directory as path,
// then renames it over path. This ensures the target file is never in a
// partially-written state if the process is interrupted.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".mcp-migrate-patch-*")
	if err != nil {
		return fmt.Errorf("patch: create temp %s: %w", dir, err)
	}
	tmpName := tmp.Name()

	fi, err := os.Stat(path)
	if err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("patch: stat %s: %w", path, err)
	}

	if err := tmp.Chmod(fi.Mode()); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("patch: chmod temp %s: %w", tmpName, err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("patch: write temp %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("patch: close temp %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("patch: rename %s -> %s: %w", tmpName, path, err)
	}
	return nil
}

// unifiedDiff produces a well-formed unified diff between old and new content.
func unifiedDiff(path string, old, new []byte) string {
	if bytes.Equal(old, new) {
		return ""
	}
	oldLines := strings.Split(string(old), "\n")
	newLines := strings.Split(string(new), "\n")

	const ctx = 3

	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}
	var changedIdx []int
	for i := 0; i < maxLen; i++ {
		if lineAt(oldLines, i) != lineAt(newLines, i) {
			changedIdx = append(changedIdx, i)
		}
	}
	if len(changedIdx) == 0 {
		return ""
	}

	type hunkRange struct{ lo, hi int }
	var hunks []hunkRange
	lo, hi := changedIdx[0], changedIdx[0]
	for _, idx := range changedIdx[1:] {
		if idx-hi <= ctx*2 {
			hi = idx
		} else {
			hunks = append(hunks, hunkRange{lo, hi})
			lo, hi = idx, idx
		}
	}
	hunks = append(hunks, hunkRange{lo, hi})

	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n", path)
	fmt.Fprintf(&b, "+++ b/%s\n", path)

	for _, hr := range hunks {
		oldWinStart := hr.lo - ctx
		if oldWinStart < 0 {
			oldWinStart = 0
		}
		oldWinEnd := hr.hi + ctx
		if oldWinEnd >= len(oldLines) {
			oldWinEnd = len(oldLines) - 1
		}
		newWinStart := hr.lo - ctx
		if newWinStart < 0 {
			newWinStart = 0
		}
		newWinEnd := hr.hi + ctx
		if newWinEnd >= len(newLines) {
			newWinEnd = len(newLines) - 1
		}

		winStart := oldWinStart
		if newWinStart < winStart {
			winStart = newWinStart
		}
		winEnd := oldWinEnd
		if newWinEnd > winEnd {
			winEnd = newWinEnd
		}

		type emitLine struct {
			kind byte
			text string
		}
		var emit []emitLine
		for i := winStart; i <= winEnd; i++ {
			ol := lineAt(oldLines, i)
			nl := lineAt(newLines, i)
			if ol == nl {
				emit = append(emit, emitLine{' ', ol})
			} else {
				if i < len(oldLines) {
					emit = append(emit, emitLine{'-', ol})
				}
				if i < len(newLines) {
					emit = append(emit, emitLine{'+', nl})
				}
			}
		}

		oldCount, newCount := 0, 0
		for _, e := range emit {
			switch e.kind {
			case ' ':
				oldCount++
				newCount++
			case '-':
				oldCount++
			case '+':
				newCount++
			}
		}

		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n",
			oldWinStart+1, oldCount, newWinStart+1, newCount)
		for _, e := range emit {
			fmt.Fprintf(&b, "%c%s\n", e.kind, e.text)
		}
	}

	return b.String()
}

// lineAt returns lines[i] or "" when i is out of bounds.
func lineAt(lines []string, i int) string {
	if i < 0 || i >= len(lines) {
		return ""
	}
	return lines[i]
}
