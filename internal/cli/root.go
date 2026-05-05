// Package cli implements the OctoJ command-line interface using Cobra.
package cli

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	verbose  bool
	logLevel string
)

// NewRootCmd creates and returns the root cobra command for OctoJ.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "octoj",
		Short: "OctoJ — Multi-platform Java JDK version manager",
		Long: `OctoJ is a multi-platform Java JDK version manager.
Manage multiple JDK versions across Temurin, Corretto, Zulu, and Liberica.

Made with love by OctavoBit — https://github.com/vituBIG/OctoJ`,
		Version: currentVersion,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(cmd)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.octoj/config.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (trace, debug, info, warn, error)")

	// Register all subcommands
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(newCurrentCmd())
	rootCmd.AddCommand(newInstalledCmd())
	rootCmd.AddCommand(newUninstallCmd())
	rootCmd.AddCommand(newEnvCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newCacheCmd())
	rootCmd.AddCommand(newSelfUpdateCmd())
	rootCmd.AddCommand(newPurgeCmd())

	return rootCmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig(cmd *cobra.Command) error {
	// Set log level
	level := zerolog.InfoLevel
	if verbose {
		level = zerolog.DebugLevel
	} else {
		switch strings.ToLower(logLevel) {
		case "trace":
			level = zerolog.TraceLevel
		case "debug":
			level = zerolog.DebugLevel
		case "info":
			level = zerolog.InfoLevel
		case "warn":
			level = zerolog.WarnLevel
		case "error":
			level = zerolog.ErrorLevel
		}
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(level)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		viper.AddConfigPath(home + "/.octoj")
		viper.SetConfigName("config")
		viper.SetConfigType("json")
	}

	viper.SetEnvPrefix("OCTOJ")
	viper.AutomaticEnv()

	// Read config — it's OK if config doesn't exist yet
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Debug().Err(err).Msg("could not read config file")
		}
	}

	return nil
}
