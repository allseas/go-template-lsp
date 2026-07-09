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

	log.Print("starting server")

	err := Init()
	if err != nil {
		log.Fatal().Err(err).Msg("error initializing handlers")
	}
}
