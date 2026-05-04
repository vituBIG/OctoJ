// Package main is the entry point for the OctoJ CLI.
// OctoJ is a multi-platform Java JDK version manager inspired by nvm/jabba/sdkman.
package main

import (
	"os"

	"github.com/OctavoBit/octoj/internal/cli"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure zerolog with console writer for human-readable output
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	rootCmd := cli.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}
