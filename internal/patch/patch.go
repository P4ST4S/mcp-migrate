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

// Apply scans Path for patchable occurrences of -32002 and, when Write is
// true, rewrites the affected files. It always returns a diff regardless of
// Write.
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

// errorCodePattern matches the literal integer -32002 as a standalone token
// (not followed by another digit).
var errorCodePattern = regexp.MustCompile(`-32002([^0-9])`)

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

	// Build the patched content line by line.
	newLines := make([][]byte, len(lines))
	copy(newLines, lines)
	for _, d := range decisions {
		if !d.safe {
			continue
		}
		// Replace -32002 preserving the trailing non-digit capture group.
		newLines[d.lineIdx] = errorCodePattern.ReplaceAll(newLines[d.lineIdx], []byte("-32602$1"))
	}
	patched := bytes.Join(newLines, []byte("\n"))

	diff := unifiedDiff(path, src, patched)

	if write {
		fi, err := os.Stat(path)
		if err != nil {
			return FileResult{}, fmt.Errorf("patch: stat %s: %w", path, err)
		}
		if err := os.WriteFile(path, patched, fi.Mode()); err != nil {
			return FileResult{}, fmt.Errorf("patch: write %s: %w", path, err)
		}
	}

	return FileResult{
		Path:    path,
		Changed: true,
		Skipped: skippedCount,
		Diff:    diff,
	}, nil
}

// unifiedDiff produces a simplified unified diff between old and new content.
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
		ol, nl := "", ""
		if i < len(oldLines) {
			ol = oldLines[i]
		}
		if i < len(newLines) {
			nl = newLines[i]
		}
		if ol != nl {
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
		hunkOldStart := hr.lo - ctx
		if hunkOldStart < 0 {
			hunkOldStart = 0
		}
		hunkOldEnd := hr.hi + ctx
		if hunkOldEnd >= len(oldLines) {
			hunkOldEnd = len(oldLines) - 1
		}
		hunkNewStart := hr.lo - ctx
		if hunkNewStart < 0 {
			hunkNewStart = 0
		}
		hunkNewEnd := hr.hi + ctx
		if hunkNewEnd >= len(newLines) {
			hunkNewEnd = len(newLines) - 1
		}

		hunkEnd := hunkOldEnd
		if hunkNewEnd > hunkEnd {
			hunkEnd = hunkNewEnd
		}

		oldCount := hunkOldEnd - hunkOldStart + 1
		newCount := hunkNewEnd - hunkNewStart + 1

		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n",
			hunkOldStart+1, oldCount, hunkNewStart+1, newCount)

		for i := hunkOldStart; i <= hunkEnd; i++ {
			ol, nl := "", ""
			if i < len(oldLines) {
				ol = oldLines[i]
			}
			if i < len(newLines) {
				nl = newLines[i]
			}
			if ol == nl {
				fmt.Fprintf(&b, " %s\n", ol)
			} else {
				if i < len(oldLines) {
					fmt.Fprintf(&b, "-%s\n", ol)
				}
				if i < len(newLines) {
					fmt.Fprintf(&b, "+%s\n", nl)
				}
			}
		}
	}

	return b.String()
}
