package patch

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Options controls a patch run.
type Options struct {
	// Path is the file or directory to patch. Required.
	Path string
	// Write commits changes to disk. When false (default) the run is dry-run:
	// diffs are produced but no file is modified.
	Write bool
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
}

// ignoredDirs are directory names that are never descended into during a walk.
var ignoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".hg":          true,
	".svn":         true,
}

// Apply scans Path for patchable occurrences of -32002 and, when Write is
// true, rewrites the affected files atomically. It always returns a diff
// regardless of Write.
func Apply(opts Options) (Result, error) {
	if opts.Path == "" {
		return Result{}, fmt.Errorf("patch: path is required")
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
// It uses a word-boundary-style assertion: the four digits must not be
// immediately preceded or followed by another digit.
// Using (?:^|[^0-9]) as a look-behind substitute and ([^0-9]|$) as look-ahead
// lets us match -32002 at end-of-line and end-of-file without requiring a
// trailing non-digit character to be consumed.
var errorCodePattern = regexp.MustCompile(`(^|[^0-9])-32002($|[^0-9])`)

// resourceContextPattern matches identifiers that, when present near -32002,
// confirm the code is used in a resource-not-found context. The pattern
// requires one of:
//   - the exact MCP method name "resources/read" (with slash)
//   - "resource not found" / "resource_not_found" as a compound phrase
//   - "ResourceNotFound" as a single camel-case identifier
//
// Plain words like "resource" or "legacyResourceCode" do not match, to prevent
// false positives in generic error-handling code.
var resourceContextPattern = regexp.MustCompile(
	`(?i)resources/read|resource[_ -]not[_ -]found|ResourceNotFound`,
)

// contextWindowLines is the number of lines above and below a -32002
// occurrence that are searched for a resource context signal.
const contextWindowLines = 12

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
	type decision struct {
		lineIdx int
		safe    bool
	}
	decisions := make([]decision, len(matchLines))
	for k, idx := range matchLines {
		lo := idx - contextWindowLines
		if lo < 0 {
			lo = 0
		}
		hi := idx + contextWindowLines
		if hi >= len(lines) {
			hi = len(lines) - 1
		}
		window := bytes.Join(lines[lo:hi+1], []byte("\n"))
		decisions[k] = decision{lineIdx: idx, safe: resourceContextPattern.Match(window)}
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
	// The pattern captures the non-digit characters surrounding -32002 so they
	// are preserved in the substitution.
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

	// Preserve permissions of the original file.
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
// Each hunk header accurately reflects the line counts of the context lines
// actually emitted, making the diff applicable with patch(1).
func unifiedDiff(path string, old, new []byte) string {
	if bytes.Equal(old, new) {
		return ""
	}
	oldLines := strings.Split(string(old), "\n")
	newLines := strings.Split(string(new), "\n")

	const ctx = 3

	// Identify which line indices differ (using the longer slice as the bound).
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}
	var changedIdx []int
	for i := 0; i < maxLen; i++ {
		ol, nl := lineAt(oldLines, i), lineAt(newLines, i)
		if ol != nl {
			changedIdx = append(changedIdx, i)
		}
	}
	if len(changedIdx) == 0 {
		return ""
	}

	// Group adjacent changes (within 2*ctx of each other) into hunks.
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
		// Compute the window of lines to emit for this hunk.
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

		// The hunk iterates over the union of both windows.
		winStart := oldWinStart
		if newWinStart < winStart {
			winStart = newWinStart
		}
		winEnd := oldWinEnd
		if newWinEnd > winEnd {
			winEnd = newWinEnd
		}

		// Collect lines to emit so we can compute accurate counts before
		// writing the @@ header.
		type emitLine struct {
			kind byte // ' ', '-', '+'
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

		// Count old and new lines from the collected emit lines.
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
