// Package main initializes and starts the Go text/template Language Server Protocol (LSP) server, setting up logging and handling any initialization errors.
package main

import (
	"os"
	"text-template-server/handlers"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	lsName  = "goTmpl"
	version = "0.1.0"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})

	log.Print("starting server")

	err := handlers.Init(lsName, version)
	if err != nil {
		log.Fatal().Err(err).Msg("error initializing handlers")
	}
}
