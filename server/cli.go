package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text-template-server/handlers"
	"text-template-server/types"

	"github.com/rs/zerolog"
	protocol "github.com/tliron/glsp/protocol_3_16"
	lspuri "go.lsp.dev/uri"
)

const checkUsage = `Usage: gotmpls check [flags] <file|glob> [<file|glob>...]

Analyze Go text/template files and print their diagnostics. Use "-" as a file
argument to read a single template from stdin.

Exit codes:
  0  no diagnostic at or above the failure threshold
  1  at least one diagnostic at or above the threshold (see --min-severity)
  2  usage error, or a file could not be read

Flags:
`

// fileDiagnostics pairs a file path with the diagnostics found in it. It is the
// JSON output element for `check --format json`.
type fileDiagnostics struct {
	Path        string                `json:"file"`
	Diagnostics []protocol.Diagnostic `json:"diagnostics"`
}

// fprintf and fprintln write best-effort diagnostics to a terminal stream,
// discarding write errors (there is nothing useful to do if stderr/stdout
// fails).
func fprintf(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...) //nolint:errcheck // best-effort stream write
}

func fprintln(w io.Writer, a ...any) {
	_, _ = fmt.Fprintln(w, a...) //nolint:errcheck // best-effort stream write
}

// runCheck implements the `check` subcommand. It returns the process exit code.
func runCheck(args []string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	root := fs.String(
		"root", "", "workspace root for type resolution (default: current directory)",
	)
	minSeverity := fs.String(
		"min-severity", "error",
		"severity that triggers a non-zero exit: error, warning, information, hint",
	)
	verbose := fs.Bool("verbose", false, "enable server debug logging on stderr")
	fs.Usage = func() {
		fprintf(stderr, "%s", checkUsage)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if !*verbose {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	threshold, ok := severityFromName(*minSeverity)
	if !ok {
		fprintf(stderr, "invalid --min-severity %q\n", *minSeverity)
		return 2
	}
	if *format != "text" && *format != "json" {
		fprintf(stderr, "invalid --format %q\n", *format)
		return 2
	}
	if fs.NArg() == 0 {
		fprintln(stderr, "no files given")
		fs.Usage()
		return 2
	}

	handlers.SetWorkspaceRoot(resolveRoot(*root))

	// Seed the global funcs cache (builtins plus workspace-defined
	// //tmpl:func "global" hints) exactly as the LSP initialize handler does,
	// so batch analysis does not report every function reference as undefined.
	funcs, err := types.ComputeGlobalFuncs(handlers.WorkspaceRoot)
	if err != nil {
		fprintf(stderr, "warning: failed to load global tmpl:func hints: %v\n", err)
	}
	types.SetGlobalFuncs(funcs)

	paths, err := expandPaths(fs.Args())
	if err != nil {
		fprintln(stderr, err)
		return 2
	}

	var results []fileDiagnostics
	readErr := false
	for _, p := range paths {
		src, serr := readSource(p, stdin)
		if serr != nil {
			fprintf(stderr, "%s: %v\n", p, serr)
			readErr = true
			continue
		}
		diags := handlers.AnalyzeDocument(src.uri, src.text)
		results = append(results, fileDiagnostics{Path: src.displayPath, Diagnostics: diags})
	}

	exceeded := renderResults(stdout, results, *format, threshold)

	switch {
	case readErr:
		return 2
	case exceeded:
		return 1
	default:
		return 0
	}
}

// source is a template to analyse together with how to display and address it.
type source struct {
	text        string
	displayPath string
	uri         string
}

// readSource loads a single template from a path or, when p is "-", from stdin.
func readSource(p string, stdin io.Reader) (source, error) {
	if p == "-" {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return source{}, err
		}
		cwd, _ := os.Getwd()
		return source{
			text:        string(b),
			displayPath: "<stdin>",
			uri:         pathToURI(filepath.Join(cwd, "stdin.tmpl")),
		}, nil
	}
	b, err := os.ReadFile(p) //nolint:gosec // CLI reads user-given paths
	if err != nil {
		return source{}, err
	}
	return source{text: string(b), displayPath: p, uri: pathToURI(p)}, nil
}

// resolveRoot returns an absolute workspace root, defaulting to the current
// working directory when root is empty.
func resolveRoot(root string) string {
	if root == "" {
		if cwd, err := os.Getwd(); err == nil {
			return cwd
		}
		return ""
	}
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

// pathToURI converts a filesystem path to an absolute file:// URI.
func pathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return string(lspuri.File(abs))
}

// expandPaths expands any glob patterns in args, preserving literal paths and
// the "-" stdin marker.
func expandPaths(args []string) ([]string, error) {
	var out []string
	for _, a := range args {
		if a == "-" || !hasGlobMeta(a) {
			out = append(out, a)
			continue
		}
		matches, err := filepath.Glob(a)
		if err != nil {
			return nil, fmt.Errorf("bad pattern %q: %w", a, err)
		}
		out = append(out, matches...)
	}
	return out, nil
}

// hasGlobMeta reports whether s contains shell-glob metacharacters.
func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// renderResults writes results in the requested format and reports whether any
// diagnostic is at or above threshold.
func renderResults(
	w io.Writer,
	results []fileDiagnostics,
	format string,
	threshold protocol.DiagnosticSeverity,
) bool {
	exceeded := false
	for _, r := range results {
		for _, d := range r.Diagnostics {
			if atOrAboveThreshold(d.Severity, threshold) {
				exceeded = true
			}
		}
	}

	if format == "json" {
		if results == nil {
			results = []fileDiagnostics{}
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			return exceeded
		}
		return exceeded
	}

	for _, r := range results {
		for _, d := range r.Diagnostics {
			fprintf(
				w, "%s:%d:%d: %s: %s\n",
				r.Path,
				d.Range.Start.Line+1,
				d.Range.Start.Character+1,
				severityName(d.Severity),
				d.Message,
			)
		}
	}
	return exceeded
}

// severityFromName maps a severity name (as passed to --min-severity) to its
// protocol value.
func severityFromName(name string) (protocol.DiagnosticSeverity, bool) {
	switch strings.ToLower(name) {
	case "error":
		return protocol.DiagnosticSeverityError, true
	case "warning", "warn":
		return protocol.DiagnosticSeverityWarning, true
	case "information", "info":
		return protocol.DiagnosticSeverityInformation, true
	case "hint":
		return protocol.DiagnosticSeverityHint, true
	default:
		return 0, false
	}
}

// severityName renders a diagnostic severity for text output.
func severityName(sev *protocol.DiagnosticSeverity) string {
	if sev == nil {
		return "unknown"
	}
	switch *sev {
	case protocol.DiagnosticSeverityError:
		return "error"
	case protocol.DiagnosticSeverityWarning:
		return "warning"
	case protocol.DiagnosticSeverityInformation:
		return "information"
	case protocol.DiagnosticSeverityHint:
		return "hint"
	default:
		return "unknown"
	}
}

// atOrAboveThreshold reports whether sev is at least as severe as threshold.
// LSP severities run from Error(1) to Hint(4), so a lower number is more
// severe and "at or above" means the numeric value is <= threshold.
func atOrAboveThreshold(
	sev *protocol.DiagnosticSeverity,
	threshold protocol.DiagnosticSeverity,
) bool {
	if sev == nil {
		return false
	}
	return *sev <= threshold
}
