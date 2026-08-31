//go:build !cli

package main

import (
	"os"

	"github.com/rs/zerolog/log"
)

// main is the entry point for the full build. With no arguments it starts the
// stdio LSP server; `check <files...>` runs batch analysis and exits.
func main() {
	setupLogging()

	// Batch mode: `gotmpls check <files...>` analyses files and exits, instead
	// of starting the stdio LSP server (the default with no arguments).
	if len(os.Args) > 1 && os.Args[1] == "check" {
		os.Exit(runCheck(os.Args[2:], os.Stdout, os.Stderr, os.Stdin))
	}

	log.Print("starting server")

	if err := Init(); err != nil {
		log.Fatal().Err(err).Msg("error initializing handlers")
	}
}
