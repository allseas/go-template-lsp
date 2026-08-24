// Package main initializes and starts the Go text/template Language Server Protocol (LSP) server, setting up logging and handling any initialization errors.
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

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})

	// Batch mode: `gotmpls check <files...>` analyses files and exits, instead
	// of starting the stdio LSP server (the default with no arguments).
	if len(os.Args) > 1 && os.Args[1] == "check" {
		os.Exit(runCheck(os.Args[2:], os.Stdout, os.Stderr, os.Stdin))
	}

	log.Print("starting server")

	err := Init()
	if err != nil {
		log.Fatal().Err(err).Msg("error initializing handlers")
	}
}
