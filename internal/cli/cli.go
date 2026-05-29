package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/P4ST4S/mcp-migrate/internal/analyze/live"
	"github.com/P4ST4S/mcp-migrate/internal/patch"
	"github.com/P4ST4S/mcp-migrate/internal/report"
	"github.com/P4ST4S/mcp-migrate/internal/spec"
)

type AnalyzeOptions struct {
	Transport           string
	URL                 string
	ServerCommand       string
	Format              string
	SpecTarget          string
	AllowMutatingProbes bool
	AllowResourceRead   bool
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printRootUsage(stderr)
		return 2
	}

	switch args[0] {
	case "analyze":
		return runAnalyze(args[1:], stdout, stderr)
	case "patch":
		return runPatch(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printRootUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printRootUsage(stderr)
		return 2
	}
}

func runAnalyze(args []string, stdout, stderr io.Writer) int {
	opts, err := parseAnalyze(args, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	findings, err := live.Analyze(live.Options{
		Transport:           opts.Transport,
		URL:                 opts.URL,
		ServerCommand:       opts.ServerCommand,
		SpecTarget:          opts.SpecTarget,
		AllowMutatingProbes: opts.AllowMutatingProbes,
		AllowResourceRead:   opts.AllowResourceRead,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	switch opts.Format {
	case "jsonl":
		if err := report.WriteJSONL(stdout, findings); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "markdown":
		if err := report.WriteMarkdown(stdout, findings); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	default:
		fmt.Fprintf(stderr, "unsupported format %q\n", opts.Format)
		return 2
	}

	return 0
}

func parseAnalyze(args []string, output io.Writer) (AnalyzeOptions, error) {
	opts := AnalyzeOptions{
		Transport:  "http",
		Format:     "jsonl",
		SpecTarget: spec.TargetVersion,
	}

	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&opts.Transport, "transport", opts.Transport, "transport to analyze: http or stdio")
	fs.StringVar(&opts.URL, "url", "", "Streamable HTTP MCP endpoint")
	fs.StringVar(&opts.ServerCommand, "server-command", "", "stdio server command")
	fs.StringVar(&opts.Format, "format", opts.Format, "output format: jsonl or markdown")
	fs.StringVar(&opts.SpecTarget, "spec-target", opts.SpecTarget, "target MCP specification version")
	fs.BoolVar(&opts.AllowMutatingProbes, "allow-mutating-probes", false, "allow probes that may modify server state")
	fs.BoolVar(&opts.AllowResourceRead, "allow-resource-read", false, "allow resources/read probes; disabled by default because reads can have server-specific side effects")

	if err := fs.Parse(args); err != nil {
		return AnalyzeOptions{}, err
	}

	switch opts.Transport {
	case "http", "stdio":
	default:
		return AnalyzeOptions{}, fmt.Errorf("unsupported transport %q", opts.Transport)
	}

	return opts, nil
}

func runPatch(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("patch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var path string
	var write, allowPending bool
	fs.StringVar(&path, "path", "", "file or directory to patch (required)")
	fs.BoolVar(&write, "write", false, "write changes to disk (default: dry-run)")
	fs.BoolVar(&allowPending, "allow-pending", false,
		"allow patches governed by rules whose SEP has not yet reached Final status; "+
			"carries the risk of a second migration if the spec changes before finalisation")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if path == "" {
		fmt.Fprintln(stderr, "patch: --path is required")
		return 2
	}

	result, err := patch.Apply(patch.Options{Path: path, Write: write, AllowPending: allowPending})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if result.PendingWarning != "" {
		fmt.Fprintln(stderr, result.PendingWarning)
	}

	if len(result.Files) == 0 {
		fmt.Fprintln(stdout, "no patchable occurrences found")
		return 0
	}

	for _, fr := range result.Files {
		if fr.Changed {
			if write {
				fmt.Fprintf(stdout, "patched %s\n", fr.Path)
			} else {
				fmt.Fprintf(stdout, "would patch %s\n", fr.Path)
			}
			if fr.Diff != "" {
				fmt.Fprintln(stdout, fr.Diff)
			}
		}
		if fr.Skipped > 0 {
			fmt.Fprintf(stdout, "skipped %d ambiguous occurrence(s) in %s\n", fr.Skipped, fr.Path)
		}
	}
	return 0
}

func printRootUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: mcp-migrate <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  analyze    analyze an MCP server and emit findings")
	fmt.Fprintln(w, "  patch      apply safe mechanical patches to source files")
}
