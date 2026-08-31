//go:build cli

package main

import "os"

// main is the entry point for the check-only build (built with `-tags cli`).
// The LSP server is not compiled in, so the binary is lighter and every
// argument is passed straight to the checker: the `check` subcommand does not
// need to be typed.
func main() {
	setupLogging()
	os.Exit(runCheck(os.Args[1:], os.Stdout, os.Stderr, os.Stdin))
}
