// Package main provides the Go text/template tooling entry points. The default
// build is the full LSP server (which also exposes the `check` subcommand). A
// lighter, check-only CLI without any LSP capabilities is produced with the
// `cli` build tag (see main_cli.go).
package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	lsName  = "goTmpl"
	version = "1.2.0"
)

// setupLogging configures the global zerolog logger to write human-readable
// output to stderr. Both build variants call it before doing any work.
func setupLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})
}
